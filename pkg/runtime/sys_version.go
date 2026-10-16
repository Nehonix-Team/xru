package runtime

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/Nehonix-Team/xru/internal/utils"
)

// parseSemVer parses semantic version strings like "v2.0.67", "2.0.67", "2.1"
func parseSemVer(v string) (major, minor, patch int, err error) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")

	// Strip pre-release or build metadata (e.g., "-beta", "+build")
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	if len(parts) >= 1 && parts[0] != "" {
		major, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid semver major: %s", parts[0])
		}
	}
	if len(parts) >= 2 && parts[1] != "" {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid semver minor: %s", parts[1])
		}
	}
	if len(parts) >= 3 && parts[2] != "" {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid semver patch: %s", parts[2])
		}
	}
	return major, minor, patch, nil
}

// compareSemVer returns:
// -1 if v1 < v2
//  0 if v1 == v2
//  1 if v1 > v2
func compareSemVer(v1, v2 string) (int, error) {
	maj1, min1, pat1, err1 := parseSemVer(v1)
	if err1 != nil {
		return 0, err1
	}
	maj2, min2, pat2, err2 := parseSemVer(v2)
	if err2 != nil {
		return 0, err2
	}

	if maj1 != maj2 {
		if maj1 < maj2 {
			return -1, nil
		}
		return 1, nil
	}
	if min1 != min2 {
		if min1 < min2 {
			return -1, nil
		}
		return 1, nil
	}
	if pat1 != pat2 {
		if pat1 < pat2 {
			return -1, nil
		}
		return 1, nil
	}
	return 0, nil
}

// checkVersionConstraint evaluates a version constraint against currentVer.
// Supported operators: >=, <=, ==, =, >, <, or plain version (defaults to >=).
func checkVersionConstraint(currentVer, constraint string) (bool, error) {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true, nil
	}

	op := ">="
	targetVer := constraint

	if strings.HasPrefix(constraint, ">=") {
		op = ">="
		targetVer = strings.TrimSpace(constraint[2:])
	} else if strings.HasPrefix(constraint, "<=") {
		op = "<="
		targetVer = strings.TrimSpace(constraint[2:])
	} else if strings.HasPrefix(constraint, "==") {
		op = "=="
		targetVer = strings.TrimSpace(constraint[2:])
	} else if strings.HasPrefix(constraint, "=") {
		op = "=="
		targetVer = strings.TrimSpace(constraint[1:])
	} else if strings.HasPrefix(constraint, ">") {
		op = ">"
		targetVer = strings.TrimSpace(constraint[1:])
	} else if strings.HasPrefix(constraint, "<") {
		op = "<"
		targetVer = strings.TrimSpace(constraint[1:])
	}

	cmp, err := compareSemVer(currentVer, targetVer)
	if err != nil {
		return false, err
	}

	switch op {
	case ">=":
		return cmp >= 0, nil
	case "<=":
		return cmp <= 0, nil
	case "==", "=":
		return cmp == 0, nil
	case ">":
		return cmp > 0, nil
	case "<":
		return cmp < 0, nil
	default:
		return false, fmt.Errorf("unknown operator '%s'", op)
	}
}

// autoUpdateXRU downloads the latest XRU binary from GitHub releases and replaces the current binary.
func autoUpdateXRU(silent bool) error {
	osName := goRuntime.GOOS
	archName := goRuntime.GOARCH

	ext := ""
	if osName == "windows" {
		ext = ".exe"
	}

	binaryFileName := fmt.Sprintf("xru-%s-%s%s", osName, archName, ext)
	downloadURL := fmt.Sprintf("https://github.com/Nehonix-Team/xru/releases/latest/download/%s", binaryFileName)

	if !silent {
		fmt.Printf("\033[36m[XRU AUTO-UPDATE]\033[0m Downloading latest XRU binary from %s...\n", downloadURL)
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to fetch latest binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	execPath, err := os.Executable()
	if err != nil {
		execPath = "xru" + ext
	}
	execPath, _ = filepath.EvalSymlinks(execPath)

	tmpFile, err := os.CreateTemp("", "xru_update_*"+ext)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission: %w", err)
	}

	// Try replacing current executable
	replaceErr := os.Rename(tmpPath, execPath)
	if replaceErr != nil {
		// Fallback: try copy if rename fails across filesystems or permissions
		in, inErr := os.Open(tmpPath)
		if inErr == nil {
			defer in.Close()
			out, outErr := os.OpenFile(execPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
			if outErr == nil {
				defer out.Close()
				_, copyErr := io.Copy(out, in)
				if copyErr == nil {
					replaceErr = nil
				}
			}
		}
	}

	if replaceErr != nil {
		// Fallback location ~/.local/bin/xru or ~/.xfpm/bin/xru
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			fallbackBin := filepath.Join(homeDir, ".xfpm", "bin", "xru"+ext)
			_ = os.MkdirAll(filepath.Dir(fallbackBin), 0755)
			if copyFile(tmpPath, fallbackBin) == nil {
				_ = os.Chmod(fallbackBin, 0755)
				execPath = fallbackBin
				replaceErr = nil
			}
		}
	}

	if replaceErr != nil {
		return fmt.Errorf("could not replace binary at %s (permission denied?): %w", execPath, replaceErr)
	}

	if !silent {
		fmt.Printf("\033[32m[XRU AUTO-UPDATE]\033[0m Successfully updated XRU at %s\n", execPath)
		fmt.Printf("\033[36m[XRU AUTO-UPDATE]\033[0m Restarting execution with updated XRU binary...\n\n")
	}

	// Restart process seamlessly
	if goRuntime.GOOS != "windows" {
		return syscall.Exec(execPath, os.Args, os.Environ())
	}

	cmd := exec.Command(execPath, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	os.Exit(0)
	return nil
}


// sysRequireFn implements Sys.require(constraint, options?)
func sysRequireFn(args []Value) (Value, error) {
	if len(args) < 1 {
		return NilValue{}, fmt.Errorf("Sys.require requires at least 1 argument (version constraint)")
	}

	constraint := args[0].String()
	onMismatch := "fatal"
	mode := "log"

	if len(args) > 1 {
		if m, ok := args[1].(MapValue); ok {
			if val, exists := m.Pairs["onMismatch"]; exists {
				onMismatch = strings.ToLower(val.String())
			}
			if val, exists := m.Pairs["mode"]; exists {
				mode = strings.ToLower(val.String())
			}
		}
	}

	currentVer := utils.BinVersion
	matched, err := checkVersionConstraint(currentVer, constraint)
	if err != nil {
		return NilValue{}, fmt.Errorf("invalid version constraint '%s': %w", constraint, err)
	}

	if matched {
		return BoolValue{Val: true}, nil
	}

	// Mismatch handling
	switch onMismatch {
	case "auto-update", "update":
		err := autoUpdateXRU(mode == "silent")
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31m[XRU AUTO-UPDATE ERROR]\033[0m %v\n", err)
			os.Exit(1)
		}
		return BoolValue{Val: true}, nil

	case "warn":
		if mode != "silent" {
			fmt.Fprintf(os.Stderr, "\033[33m[XRU VERSION WARNING]\033[0m This script requires XRU %s, currently running %s.\n", constraint, currentVer)
		}
		return BoolValue{Val: false}, nil

	default: // "fatal"
		fmt.Fprintf(os.Stderr, "\n\033[1;31m[XRU VERSION ERROR]\033[0m This script requires XRU \033[1m%s\033[0m (current version: \033[33m%s\033[0m).\n", constraint, currentVer)
		fmt.Fprintf(os.Stderr, "To update XRU to the latest version:\n")
		fmt.Fprintf(os.Stderr, "  \033[36m• curl -fsSL https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.sh | bash -s -- xru\033[0m\n")
		fmt.Fprintf(os.Stderr, "  \033[36m• or download from: https://github.com/Nehonix-Team/xru/releases/latest\033[0m\n\n")
		os.Exit(1)
		return BoolValue{Val: false}, nil
	}
}

// sysVersionFn implements Sys.version()
func sysVersionFn(args []Value) (Value, error) {
	return StringValue{Val: utils.BinVersion}, nil
}
