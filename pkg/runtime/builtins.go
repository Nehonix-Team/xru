package runtime

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	mathRand "math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goRuntime "runtime"
	"strconv"
	"strings"
	"time"
)

func NewBuiltinUnits() map[string]*UnitInstance {
	units := make(map[string]*UnitInstance)

	// 1. #UNIT std (Standard Library I/O & Output)
	printlnFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			fmt.Println()
			return NilValue{}, nil
		}
		msg := formatTerminalColors(args[0].String())
		fmt.Println(msg)
		return NilValue{}, nil
	}
	printFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		msg := formatTerminalColors(args[0].String())
		fmt.Print(msg)
		return NilValue{}, nil
	}
	infoFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		fmt.Println(formatTerminalColors("<cyan>[INFO]</> " + args[0].String()))
		return NilValue{}, nil
	}
	successFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		fmt.Println(formatTerminalColors("<green>[DONE]</> " + args[0].String()))
		return NilValue{}, nil
	}
	warnFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		fmt.Println(formatTerminalColors("<yellow>[WARN]</> " + args[0].String()))
		return NilValue{}, nil
	}
	errorFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		fmt.Println(formatTerminalColors("<red>[ERROR]</> " + args[0].String()))
		return NilValue{}, nil
	}

	sleepFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("sleep requires duration")
		}
		if dur, ok := args[0].(DurationValue); ok {
			time.Sleep(dur.Val)
			return BoolValue{Val: true}, nil
		}
		if num, ok := args[0].(NumberValue); ok {
			time.Sleep(time.Duration(num.Val) * time.Millisecond)
			return BoolValue{Val: true}, nil
		}
		if parsed, err := time.ParseDuration(args[0].String()); err == nil {
			time.Sleep(parsed)
			return BoolValue{Val: true}, nil
		}
		return BoolValue{Val: true}, nil
	}

	// exitCodeArg extrait un code de sortie optionnel à l'index donné.
	exitCodeArg := func(args []Value, idx int, def int) int {
		if len(args) > idx {
			if n, ok := args[idx].(NumberValue); ok {
				return int(n.Val)
			}
		}
		return def
	}

	// fatalFn : sortie brutale non rattrapable (équivalent de panic/throw non capturé).
	fatalFn := func(args []Value) (Value, error) {
		msg := ""
		if len(args) > 0 {
			msg = args[0].String()
		}
		fmt.Fprintln(os.Stderr, formatTerminalColors("<red>[FATAL]</> "+msg))
		code := exitCodeArg(args, 1, 1)
		TriggerExitHooks(code)
		os.Exit(code)
		return NilValue{}, nil
	}

	// exitFn : arrêt immédiat du processus sans message.
	exitFn := func(args []Value) (Value, error) {
		code := exitCodeArg(args, 0, 0)
		TriggerExitHooks(code)
		os.Exit(code)
		return NilValue{}, nil
	}

	typeFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, fmt.Errorf("std.type requires 1 argument")
		}
		return StringValue{Val: string(args[0].Type())}, nil
	}

	terminalReader := bufio.NewReader(os.Stdin)

	inputFn := func(args []Value) (Value, error) {
		prompt := ""
		defVal := ""
		if len(args) > 0 {
			prompt = args[0].String()
		}
		if len(args) > 1 {
			defVal = args[1].String()
		}

		if prompt != "" {
			fmt.Print(formatTerminalColors(prompt))
		}
		line, err := terminalReader.ReadString('\n')
		if err != nil && len(line) == 0 {
			return StringValue{Val: defVal}, nil
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" && defVal != "" {
			return StringValue{Val: defVal}, nil
		}
		return StringValue{Val: line}, nil
	}

	selectFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("select requires at least (prompt, options)")
		}
		prompt := args[0].String()
		var options []string
		if arr, ok := args[1].(ListValue); ok {
			for _, item := range arr.Elements {
				options = append(options, item.String())
			}
		} else {
			return NilValue{}, fmt.Errorf("select: second argument must be a list of options")
		}

		if len(options) == 0 {
			return StringValue{Val: ""}, nil
		}

		defaultIdx := 0
		if len(args) > 2 {
			if n, ok := args[2].(NumberValue); ok {
				defaultIdx = int(n.Val)
			}
		}

		if prompt != "" {
			fmt.Println(formatTerminalColors(prompt))
		}
		for i, opt := range options {
			displayOpt := opt
			if displayOpt == "" {
				displayOpt = "(empty)"
			}
			marker := "  "
			if i == defaultIdx {
				marker = "* "
			}
			fmt.Println(formatTerminalColors(fmt.Sprintf("  <cyan>%s%d)</cyan> %s", marker, i+1, displayOpt)))
		}

		fmt.Print(formatTerminalColors(fmt.Sprintf("<dim>Select [1-%d] (default: %d): </dim>", len(options), defaultIdx+1)))
		line, err := terminalReader.ReadString('\n')
		if err != nil && len(line) == 0 {
			if defaultIdx >= 0 && defaultIdx < len(options) {
				return StringValue{Val: options[defaultIdx]}, nil
			}
			return StringValue{Val: ""}, nil
		}
		line = strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		if line == "" {
			if defaultIdx >= 0 && defaultIdx < len(options) {
				return StringValue{Val: options[defaultIdx]}, nil
			}
			return StringValue{Val: ""}, nil
		}

		if num, err := strconv.Atoi(line); err == nil {
			if num >= 1 && num <= len(options) {
				return StringValue{Val: options[num-1]}, nil
			}
		}

		for _, opt := range options {
			if strings.EqualFold(opt, line) {
				return StringValue{Val: opt}, nil
			}
		}

		if defaultIdx >= 0 && defaultIdx < len(options) {
			return StringValue{Val: options[defaultIdx]}, nil
		}
		return StringValue{Val: line}, nil
	}

	confirmFn := func(args []Value) (Value, error) {
		prompt := ""
		defVal := true
		if len(args) > 0 {
			prompt = args[0].String()
		}
		if len(args) > 1 {
			if b, ok := args[1].(BoolValue); ok {
				defVal = b.Val
			}
		}

		hint := "(Y/n)"
		if !defVal {
			hint = "(y/N)"
		}

		if prompt != "" {
			fmt.Print(formatTerminalColors(fmt.Sprintf("%s <dim>%s</dim> ", prompt, hint)))
		} else {
			fmt.Print(formatTerminalColors(fmt.Sprintf("<dim>%s</dim> ", hint)))
		}

		line, err := terminalReader.ReadString('\n')
		if err != nil && len(line) == 0 {
			return BoolValue{Val: defVal}, nil
		}
		line = strings.TrimSpace(strings.ToLower(strings.TrimRight(line, "\r\n")))
		if line == "" {
			return BoolValue{Val: defVal}, nil
		}
		if line == "y" || line == "yes" || line == "o" || line == "oui" || line == "1" || line == "true" {
			return BoolValue{Val: true}, nil
		}
		if line == "n" || line == "no" || line == "non" || line == "0" || line == "false" {
			return BoolValue{Val: false}, nil
		}
		return BoolValue{Val: defVal}, nil
	}

	stdUnit := &UnitInstance{
		UnitName:   "std",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"log":     printlnFn,
			"Log":     printlnFn,
			"println": printlnFn,
			"Println": printlnFn,
			"print":   printFn,
			"Print":   printFn,
			"input":   inputFn,
			"Input":   inputFn,
			"select":  selectFn,
			"Select":  selectFn,
			"confirm": confirmFn,
			"Confirm": confirmFn,
			"info":    infoFn,
			"Info":    infoFn,
			"success": successFn,
			"Success": successFn,
			"warn":    warnFn,
			"Warn":    warnFn,
			"error":   errorFn,
			"Error":   errorFn,
			"sleep":   sleepFn,
			"Sleep":   sleepFn,
			"fatal":   fatalFn,
			"Fatal":   fatalFn,
			"exit":    exitFn,
			"Exit":    exitFn,
			"type":    typeFn,
			"Type":    typeFn,
			"typeof":  typeFn,
			"Typeof":  typeFn,
		},
		Directives: make(map[string]NativeDirective),
	}
	units["std"] = stdUnit
	units["Std"] = stdUnit

	// 2. #UNIT Fs
	ensureDirFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.ensureDir requires directory path")
		}
		target := args[0].String()
		if err := os.MkdirAll(target, 0755); err != nil {
			return NilValue{}, fmt.Errorf("Fs.ensureDir failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}
	removeFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.remove requires target path")
		}
		target := args[0].String()
		if _, err := os.Lstat(target); err != nil {
			return NilValue{}, fmt.Errorf("Fs.remove failed: %w", err)
		}
		if err := os.RemoveAll(target); err != nil {
			return NilValue{}, fmt.Errorf("Fs.remove failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}
	moveFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Fs.move requires src and dest path")
		}
		src, dest := args[0].String(), args[1].String()
		if err := os.Rename(src, dest); err != nil {
			return NilValue{}, fmt.Errorf("Fs.move failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}
	copyFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Fs.copy requires src and dest path")
		}
		src, dest := args[0].String(), args[1].String()
		if err := copyPath(src, dest); err != nil {
			return NilValue{}, fmt.Errorf("Fs.copy failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}
	existsFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return BoolValue{Val: false}, nil
		}
		_, err := os.Stat(args[0].String())
		return BoolValue{Val: err == nil}, nil
	}
	readFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.read requires file path")
		}
		target := args[0].String()
		data, err := os.ReadFile(target)
		if err != nil {
			return NilValue{}, fmt.Errorf("Fs.read failed: %w", err)
		}
		return StringValue{Val: string(data)}, nil
	}
	writeFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Fs.write requires file path and content")
		}
		target := args[0].String()
		content := args[1].String()
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return NilValue{}, fmt.Errorf("Fs.write failed to create directory: %w", err)
		}
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return NilValue{}, fmt.Errorf("Fs.write failed: %w", err)
		}
		return BoolValue{Val: true}, nil
	}
	isDirFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.isDir requires target path")
		}
		info, err := os.Stat(args[0].String())
		if err != nil {
			if os.IsNotExist(err) {
				return BoolValue{Val: false}, nil
			}
			return NilValue{}, fmt.Errorf("Fs.isDir failed: %w", err)
		}
		return BoolValue{Val: info.IsDir()}, nil
	}
	isFileFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.isFile requires target path")
		}
		info, err := os.Stat(args[0].String())
		if err != nil {
			if os.IsNotExist(err) {
				return BoolValue{Val: false}, nil
			}
			return NilValue{}, fmt.Errorf("Fs.isFile failed: %w", err)
		}
		return BoolValue{Val: !info.IsDir()}, nil
	}
	sizeFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Fs.size requires file path")
		}
		info, err := os.Stat(args[0].String())
		if err != nil {
			return NilValue{}, fmt.Errorf("Fs.size failed: %w", err)
		}
		return NumberValue{Val: float64(info.Size())}, nil
	}

	// 2. #UNIT Fs
	fsUnit := &UnitInstance{
		UnitName:   "Fs",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"EnsureDir": ensureDirFn,
			"ensureDir": ensureDirFn,
			"ensuredir": ensureDirFn,
			"Remove":    removeFn,
			"remove":    removeFn,
			"Move":      moveFn,
			"move":      moveFn,
			"Copy":      copyFn,
			"copy":      copyFn,
			"Exists":    existsFn,
			"exists":    existsFn,
			"Read":      readFn,
			"read":      readFn,
			"Write":     writeFn,
			"write":     writeFn,
			"IsDir":     isDirFn,
			"isDir":     isDirFn,
			"isdir":     isDirFn,
			"IsFile":    isFileFn,
			"isFile":    isFileFn,
			"isfile":    isFileFn,
			"Size":      sizeFn,
			"size":      sizeFn,
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Fs"] = fsUnit
	units["fs"] = fsUnit

	// 3. #UNIT Net
	fetchFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Net.Fetch requires url")
		}
		url := args[0].String()
		if len(args) >= 2 {
			dest := args[1].String()
			resp, err := http.Get(url)
			if err != nil {
				return BoolValue{Val: false}, err
			}
			defer resp.Body.Close()

			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return BoolValue{Val: false}, err
			}

			out, err := os.Create(dest)
			if err != nil {
				return BoolValue{Val: false}, err
			}
			defer out.Close()

			_, err = io.Copy(out, resp.Body)
			return BoolValue{Val: err == nil}, err
		}

		resp, err := http.Get(url)
		if err != nil {
			return NilValue{}, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return NilValue{}, fmt.Errorf("HTTP error: %s", resp.Status)
		}
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return NilValue{}, err
		}
		return StringValue{Val: string(bodyBytes)}, nil
	}
	var netUnit *UnitInstance
	cloneFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("Net.Clone requires repo url and dest")
		}
		repo := args[0].String()
		dest := args[1].String()
		if !filepath.IsAbs(dest) && netUnit != nil && netUnit.WorkingDir != "" {
			dest = filepath.Join(netUnit.WorkingDir, dest)
		}

		cmd := exec.Command("git", "clone", "--depth", "1", repo, dest)
		if netUnit != nil && netUnit.WorkingDir != "" {
			cmd.Dir = netUnit.WorkingDir
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		return BoolValue{Val: err == nil}, err
	}

	netUnit = &UnitInstance{
		UnitName:   "Net",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Fetch": fetchFn,
			"fetch": fetchFn,
			"Clone": cloneFn,
			"clone": cloneFn,
		},
		Directives: map[string]NativeDirective{},
	}
	units["Net"] = netUnit
	units["net"] = netUnit

	// 4. #UNIT Sys
	var sysUnit *UnitInstance
	cwdFn := func(args []Value) (Value, error) {
		wd, err := os.Getwd()
		if err != nil {
			return StringValue{Val: ""}, err
		}
		if abs, err := filepath.Abs(wd); err == nil {
			return StringValue{Val: abs}, nil
		}
		return StringValue{Val: wd}, nil
	}

	sysUnit = &UnitInstance{
		UnitName:   "Sys",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Env": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return StringValue{Val: ""}, nil
				}
				return StringValue{Val: os.Getenv(args[0].String())}, nil
			},
			"env": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return StringValue{Val: ""}, nil
				}
				return StringValue{Val: os.Getenv(args[0].String())}, nil
			},
			"getEnv": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return StringValue{Val: ""}, nil
				}
				val := os.Getenv(args[0].String())
				if val == "" && len(args) > 1 {
					val = args[1].String()
				}
				return StringValue{Val: val}, nil
			},
			"GetEnv": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return StringValue{Val: ""}, nil
				}
				val := os.Getenv(args[0].String())
				if val == "" && len(args) > 1 {
					val = args[1].String()
				}
				return StringValue{Val: val}, nil
			},
			"Time": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format(time.RFC3339)}, nil
			},
			"time": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format(time.RFC3339)}, nil
			},
			"Os": func(args []Value) (Value, error) {
				return StringValue{Val: goRuntime.GOOS}, nil
			},
			"os": func(args []Value) (Value, error) {
				return StringValue{Val: goRuntime.GOOS}, nil
			},
			"Arch": func(args []Value) (Value, error) {
				return StringValue{Val: goRuntime.GOARCH}, nil
			},
			"arch": func(args []Value) (Value, error) {
				return StringValue{Val: goRuntime.GOARCH}, nil
			},
			"Sleep":   sleepFn,
			"sleep":   sleepFn,
			"Exit":    exitFn,
			"exit":    exitFn,
			"require": sysRequireFn,
			"Require": sysRequireFn,
			"version": sysVersionFn,
			"Version": sysVersionFn,
			"cwd":     cwdFn,
			"Cwd":     cwdFn,
			"homeDir": func(args []Value) (Value, error) {
				h, err := os.UserHomeDir()
				if err != nil {
					return StringValue{Val: ""}, err
				}
				return StringValue{Val: h}, nil
			},
			"HomeDir": func(args []Value) (Value, error) {
				h, err := os.UserHomeDir()
				if err != nil {
					return StringValue{Val: ""}, err
				}
				return StringValue{Val: h}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Sys"] = sysUnit
	units["sys"] = sysUnit

	// 5. #UNIT Cmd (exécution de commandes terminal cross-platform)
	var cmdUnit *UnitInstance
	cmdExecFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("Cmd.Exec requires command string")
		}
		commandLine := args[0].String()
		var cmd *exec.Cmd
		if goRuntime.GOOS == "windows" {
			cmd = exec.Command("cmd.exe", "/c", commandLine)
		} else {
			cmd = exec.Command("sh", "-c", commandLine)
		}

		workDir := ""
		if cmdUnit != nil && cmdUnit.WorkingDir != "" {
			workDir = cmdUnit.WorkingDir
		}
		if len(args) > 1 && args[1].String() != "" {
			customDir := args[1].String()
			if !filepath.IsAbs(customDir) && workDir != "" {
				workDir = filepath.Join(workDir, customDir)
			} else {
				workDir = customDir
			}
		}
		if workDir != "" {
			cmd.Dir = workDir
		}

		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err := cmd.Run()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return StringValue{Val: stdoutBuf.String()}, fmt.Errorf("command '%s' exited with code %d", commandLine, exitErr.ExitCode())
			}
			return StringValue{Val: stdoutBuf.String()}, err
		}
		return StringValue{Val: stdoutBuf.String()}, nil
	}

	cmdUnit = &UnitInstance{
		UnitName:   "Cmd",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Exec": cmdExecFn,
			"exec": cmdExecFn,
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Cmd"] = cmdUnit
	units["cmd"] = cmdUnit

	// 6. #UNIT Crypto (génération cryptographique sécurisée)
	cryptoUnit := &UnitInstance{
		UnitName:   "Crypto",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"GetRandomBytes": func(args []Value) (Value, error) {
				length := 32
				if len(args) > 0 {
					if n, ok := args[0].(NumberValue); ok {
						length = int(n.Val)
					}
				}
				hexStr, err := generateSecureRandomBytes(length)
				if err != nil {
					return NilValue{}, err
				}
				return StringValue{Val: hexStr}, nil
			},
			"RandomBytes": func(args []Value) (Value, error) {
				length := 32
				if len(args) > 0 {
					if n, ok := args[0].(NumberValue); ok {
						length = int(n.Val)
					}
				}
				hexStr, err := generateSecureRandomBytes(length)
				if err != nil {
					return NilValue{}, err
				}
				return StringValue{Val: hexStr}, nil
			},
			"RandomHex": func(args []Value) (Value, error) {
				length := 32
				if len(args) > 0 {
					if n, ok := args[0].(NumberValue); ok {
						length = int(n.Val)
					}
				}
				hexStr, err := generateSecureRandomBytes(length)
				if err != nil {
					return NilValue{}, err
				}
				return StringValue{Val: hexStr}, nil
			},
			"RandomBase64": func(args []Value) (Value, error) {
				length := 32
				if len(args) > 0 {
					if n, ok := args[0].(NumberValue); ok {
						length = int(n.Val)
					}
				}
				if length <= 0 {
					length = 32
				}
				b := make([]byte, length)
				if _, err := rand.Read(b); err != nil {
					return NilValue{}, err
				}
				return StringValue{Val: base64.StdEncoding.EncodeToString(b)}, nil
			},
			"Sha256": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Crypto.Sha256 requires input string")
				}
				h := sha256.Sum256([]byte(args[0].String()))
				return StringValue{Val: hex.EncodeToString(h[:])}, nil
			},
			"Uuid": func(args []Value) (Value, error) {
				b := make([]byte, 16)
				if _, err := rand.Read(b); err != nil {
					return NilValue{}, err
				}
				b[6] = (b[6] & 0x0f) | 0x40
				b[8] = (b[8] & 0x3f) | 0x80
				uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
					b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
				return StringValue{Val: uuid}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Crypto"] = cryptoUnit

	// 7. #UNIT Json
	jsonUnit := &UnitInstance{
		UnitName:   "Json",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Parse": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Json.Parse requires raw string")
				}
				raw := args[0].String()
				var parsed any
				if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
					return NilValue{}, fmt.Errorf("Json.Parse failed: %w", err)
				}
				return goValueToXruValue(parsed), nil
			},
			"Stringify": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Json.Stringify requires 1 argument")
				}
				goVal := xruValueToGoValue(args[0])
				b, err := json.Marshal(goVal)
				if err != nil {
					return NilValue{}, fmt.Errorf("Json.Stringify failed: %w", err)
				}
				return StringValue{Val: string(b)}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Json"] = jsonUnit

	// 8. #UNIT Math
	mathUnit := &UnitInstance{
		UnitName:   "Math",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Round": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Math.Round requires 1 number argument, got 0")
				}
				n, ok := args[0].(NumberValue)
				if !ok {
					return NilValue{}, fmt.Errorf("Math.Round expects number argument, got %s", args[0].Type())
				}
				return NumberValue{Val: math.Round(n.Val)}, nil
			},
			"Floor": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Math.Floor requires 1 number argument, got 0")
				}
				n, ok := args[0].(NumberValue)
				if !ok {
					return NilValue{}, fmt.Errorf("Math.Floor expects number argument, got %s", args[0].Type())
				}
				return NumberValue{Val: math.Floor(n.Val)}, nil
			},
			"Ceil": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Math.Ceil requires 1 number argument, got 0")
				}
				n, ok := args[0].(NumberValue)
				if !ok {
					return NilValue{}, fmt.Errorf("Math.Ceil expects number argument, got %s", args[0].Type())
				}
				return NumberValue{Val: math.Ceil(n.Val)}, nil
			},
			"Abs": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Math.Abs requires 1 number argument, got 0")
				}
				n, ok := args[0].(NumberValue)
				if !ok {
					return NilValue{}, fmt.Errorf("Math.Abs expects number argument, got %s", args[0].Type())
				}
				return NumberValue{Val: math.Abs(n.Val)}, nil
			},
			"Min": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("Math.Min requires 2 numbers")
				}
				a, ok1 := args[0].(NumberValue)
				b, ok2 := args[1].(NumberValue)
				if !ok1 || !ok2 {
					return NilValue{}, fmt.Errorf("Math.Min expects number arguments")
				}
				return NumberValue{Val: math.Min(a.Val, b.Val)}, nil
			},
			"Max": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("Math.Max requires 2 numbers")
				}
				a, ok1 := args[0].(NumberValue)
				b, ok2 := args[1].(NumberValue)
				if !ok1 || !ok2 {
					return NilValue{}, fmt.Errorf("Math.Max expects number arguments")
				}
				return NumberValue{Val: math.Max(a.Val, b.Val)}, nil
			},
			"Random": func(args []Value) (Value, error) {
				return NumberValue{Val: mathRand.Float64()}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Math"] = mathUnit

	// 9. #UNIT Path
	pathUnit := &UnitInstance{
		UnitName:   "Path",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Join": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Path.Join requires at least 1 path argument, got 0")
				}
				var parts []string
				for _, a := range args {
					parts = append(parts, a.String())
				}
				return StringValue{Val: filepath.Join(parts...)}, nil
			},
			"Dir": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Path.Dir requires 1 string argument, got 0")
				}
				return StringValue{Val: filepath.Dir(args[0].String())}, nil
			},
			"Base": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Path.Base requires 1 string argument, got 0")
				}
				return StringValue{Val: filepath.Base(args[0].String())}, nil
			},
			"Ext": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Path.Ext requires 1 string argument, got 0")
				}
				return StringValue{Val: filepath.Ext(args[0].String())}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Path"] = pathUnit
	units["path"] = pathUnit

	// 10. #UNIT date (Date & Time manipulation utilities)

	var dateUnit *UnitInstance
	dateUnit = &UnitInstance{
		UnitName:   "date",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"now": func(args []Value) (Value, error) {
				return dateToMap(time.Now()), nil
			},
			"Now": func(args []Value) (Value, error) {
				return dateToMap(time.Now()), nil
			},
			"iso": func(args []Value) (Value, error) {
				t := time.Now()
				if len(args) > 0 {
					var err error
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return StringValue{Val: t.Format(time.RFC3339)}, nil
			},
			"Iso": func(args []Value) (Value, error) {
				t := time.Now()
				if len(args) > 0 {
					var err error
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return StringValue{Val: t.Format(time.RFC3339)}, nil
			},
			"today": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format("2006-01-02")}, nil
			},
			"Today": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format("2006-01-02")}, nil
			},
			"timestamp": func(args []Value) (Value, error) {
				t := time.Now()
				if len(args) > 0 {
					var err error
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.UnixMilli())}, nil
			},
			"Timestamp": func(args []Value) (Value, error) {
				t := time.Now()
				if len(args) > 0 {
					var err error
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.UnixMilli())}, nil
			},
			"format": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return StringValue{Val: time.Now().Format("2006-01-02 15:04:05")}, nil
				}
				if len(args) == 1 {
					// format("YYYY-MM-DD") sur now
					fmtStr := convertDateFormat(args[0].String())
					return StringValue{Val: time.Now().Format(fmtStr)}, nil
				}
				t, err := parseTimeToTime(args[0])
				if err != nil {
					return NilValue{}, err
				}
				fmtStr := convertDateFormat(args[1].String())
				return StringValue{Val: t.Format(fmtStr)}, nil
			},
			"Format": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return StringValue{Val: time.Now().Format("2006-01-02 15:04:05")}, nil
				}
				if len(args) == 1 {
					fmtStr := convertDateFormat(args[0].String())
					return StringValue{Val: time.Now().Format(fmtStr)}, nil
				}
				t, err := parseTimeToTime(args[0])
				if err != nil {
					return NilValue{}, err
				}
				fmtStr := convertDateFormat(args[1].String())
				return StringValue{Val: t.Format(fmtStr)}, nil
			},
			"parse": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("date.parse requires date string argument")
				}
				str := args[0].String()
				if len(args) >= 2 {
					fmtStr := convertDateFormat(args[1].String())
					t, err := time.Parse(fmtStr, str)
					if err != nil {
						return NilValue{}, fmt.Errorf("date.parse failed: %w", err)
					}
					return dateToMap(t), nil
				}
				t, err := parseTimeToTime(StringValue{Val: str})
				if err != nil {
					return NilValue{}, err
				}
				return dateToMap(t), nil
			},
			"Parse": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("date.parse requires date string argument")
				}
				str := args[0].String()
				if len(args) >= 2 {
					fmtStr := convertDateFormat(args[1].String())
					t, err := time.Parse(fmtStr, str)
					if err != nil {
						return NilValue{}, fmt.Errorf("date.parse failed: %w", err)
					}
					return dateToMap(t), nil
				}
				t, err := parseTimeToTime(StringValue{Val: str})
				if err != nil {
					return NilValue{}, err
				}
				return dateToMap(t), nil
			},
			"add": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("date.add requires at least 2 arguments (date, amount, [unit])")
				}
				t, err := parseTimeToTime(args[0])
				if err != nil {
					return NilValue{}, err
				}
				if len(args) == 2 {
					// date.add(date, "2h") ou date.add(date, 3600)
					if durVal, ok := args[1].(DurationValue); ok {
						return dateToMap(t.Add(durVal.Val)), nil
					}
					if sVal, ok := args[1].(StringValue); ok {
						d, err := time.ParseDuration(sVal.Val)
						if err == nil {
							return dateToMap(t.Add(d)), nil
						}
					}
					if nVal, ok := args[1].(NumberValue); ok {
						return dateToMap(t.Add(time.Duration(nVal.Val) * time.Millisecond)), nil
					}
				}
				if len(args) >= 3 {
					amount := 0.0
					if nVal, ok := args[1].(NumberValue); ok {
						amount = nVal.Val
					}
					unit := strings.ToLower(args[2].String())
					switch unit {
					case "ms", "millisecond", "milliseconds":
						t = t.Add(time.Duration(amount) * time.Millisecond)
					case "s", "sec", "second", "seconds":
						t = t.Add(time.Duration(amount) * time.Second)
					case "m", "min", "minute", "minutes":
						t = t.Add(time.Duration(amount) * time.Minute)
					case "h", "hour", "hours":
						t = t.Add(time.Duration(amount) * time.Hour)
					case "d", "day", "days":
						t = t.AddDate(0, 0, int(amount))
					case "w", "week", "weeks":
						t = t.AddDate(0, 0, int(amount*7))
					case "month", "months":
						t = t.AddDate(0, int(amount), 0)
					case "y", "year", "years":
						t = t.AddDate(int(amount), 0, 0)
					default:
						return NilValue{}, fmt.Errorf("date.add: unknown unit '%s'", unit)
					}
					return dateToMap(t), nil
				}
				return dateToMap(t), nil
			},
			"Add": func(args []Value) (Value, error) {
				return dateUnit.Methods["add"](args)
			},
			"diff": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("date.diff requires 2 date arguments (dateA, dateB, [unit])")
				}
				tA, err := parseTimeToTime(args[0])
				if err != nil {
					return NilValue{}, err
				}
				tB, err := parseTimeToTime(args[1])
				if err != nil {
					return NilValue{}, err
				}
				diffDur := tA.Sub(tB)
				unit := "seconds"
				if len(args) >= 3 {
					unit = strings.ToLower(args[2].String())
				}
				switch unit {
				case "ms", "millisecond", "milliseconds":
					return NumberValue{Val: float64(diffDur.Milliseconds())}, nil
				case "s", "sec", "second", "seconds":
					return NumberValue{Val: diffDur.Seconds()}, nil
				case "m", "min", "minute", "minutes":
					return NumberValue{Val: diffDur.Minutes()}, nil
				case "h", "hour", "hours":
					return NumberValue{Val: diffDur.Hours()}, nil
				case "d", "day", "days":
					return NumberValue{Val: diffDur.Hours() / 24.0}, nil
				default:
					return NumberValue{Val: diffDur.Seconds()}, nil
				}
			},
			"Diff": func(args []Value) (Value, error) {
				return dateUnit.Methods["diff"](args)
			},
			"year": func(args []Value) (Value, error) {
				t, err := parseTimeToTime(nil)
				if len(args) > 0 {
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.Year())}, nil
			},
			"Year": func(args []Value) (Value, error) {
				return dateUnit.Methods["year"](args)
			},
			"month": func(args []Value) (Value, error) {
				t, err := parseTimeToTime(nil)
				if len(args) > 0 {
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.Month())}, nil
			},
			"Month": func(args []Value) (Value, error) {
				return dateUnit.Methods["month"](args)
			},
			"day": func(args []Value) (Value, error) {
				t, err := parseTimeToTime(nil)
				if len(args) > 0 {
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.Day())}, nil
			},
			"Day": func(args []Value) (Value, error) {
				return dateUnit.Methods["day"](args)
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Date"] = dateUnit
	units["date"] = dateUnit

	// 12. #UNIT sig (Signal handling & process lifecycle)
	sigUnit := &UnitInstance{
		UnitName: "Sig",
		Properties: map[string]Value{
			"interrupt": StringValue{Val: "interrupt"},
			"terminate": StringValue{Val: "terminate"},
			"hangup":    StringValue{Val: "hangup"},
			"exit":      StringValue{Val: "exit"},
			"error":     StringValue{Val: "error"},
		},
		Methods:    make(map[string]NativeMethod),
		Directives: make(map[string]NativeDirective),
	}
	units["Sig"] = sigUnit
	units["sig"] = sigUnit

	return units
}

func generateSecureRandomBytes(length int) (string, error) {
	if length <= 0 {
		length = 32
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secure random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

var reTerminalTag = regexp.MustCompile(`</?[a-zA-Z0-9_\-]*>`)

var terminalColorCodes = map[string]string{
	// Standard foreground colors
	"red":            "\033[31m",
	"green":          "\033[32m",
	"yellow":         "\033[33m",
	"blue":           "\033[34m",
	"magenta":        "\033[35m",
	"purple":         "\033[35m",
	"cyan":           "\033[36m",
	"white":          "\033[37m",
	"gray":           "\033[90m",
	"grey":           "\033[90m",
	"black":          "\033[30m",

	// Bright foreground colors
	"bright_red":     "\033[91m",
	"light_red":      "\033[91m",
	"bright_green":   "\033[92m",
	"light_green":    "\033[92m",
	"bright_yellow":  "\033[93m",
	"light_yellow":   "\033[93m",
	"bright_blue":    "\033[94m",
	"light_blue":     "\033[94m",
	"bright_magenta": "\033[95m",
	"light_magenta":  "\033[95m",
	"bright_cyan":    "\033[96m",
	"light_cyan":     "\033[96m",
	"bright_white":   "\033[97m",

	// Styles
	"bold":           "\033[1m",
	"b":              "\033[1m",
	"dim":            "\033[2m",
	"italic":         "\033[3m",
	"i":              "\033[3m",
	"underline":      "\033[4m",
	"u":              "\033[4m",

	// Backgrounds
	"bg_red":         "\033[41m",
	"bg_green":       "\033[42m",
	"bg_yellow":      "\033[43m",
	"bg_blue":        "\033[44m",
	"bg_magenta":     "\033[45m",
	"bg_cyan":        "\033[46m",
	"bg_white":       "\033[47m",
	"bg_black":       "\033[40m",
}

func formatTerminalColors(s string) string {
	if !strings.Contains(s, "<") {
		return s
	}

	matches := reTerminalTag.FindAllStringIndex(s, -1)
	if len(matches) == 0 {
		return s
	}

	var sb strings.Builder
	var stack []string // holds active tags
	lastIdx := 0

	for _, loc := range matches {
		start, end := loc[0], loc[1]
		sb.WriteString(s[lastIdx:start])
		lastIdx = end

		tagText := s[start:end]
		inner := strings.ToLower(tagText[1 : len(tagText)-1])

		if inner == "/" {
			// Generic close tag: </>
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			sb.WriteString("\033[0m")
			for _, t := range stack {
				if code, ok := terminalColorCodes[t]; ok {
					sb.WriteString(code)
				}
			}
		} else if strings.HasPrefix(inner, "/") {
			// Named close tag: </blue>, </red>, </bold>, etc.
			tagName := strings.TrimPrefix(inner, "/")
			tagName = strings.ReplaceAll(tagName, "-", "_")

			foundIdx := -1
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i] == tagName {
					foundIdx = i
					break
				}
			}
			if foundIdx != -1 {
				stack = append(stack[:foundIdx], stack[foundIdx+1:]...)
			}

			sb.WriteString("\033[0m")
			for _, t := range stack {
				if code, ok := terminalColorCodes[t]; ok {
					sb.WriteString(code)
				}
			}
		} else {
			// Opening tag: <blue>, <red>, etc.
			tagName := strings.ReplaceAll(inner, "-", "_")
			if code, ok := terminalColorCodes[tagName]; ok {
				stack = append(stack, tagName)
				sb.WriteString(code)
			} else {
				sb.WriteString(tagText)
			}
		}
	}

	sb.WriteString(s[lastIdx:])
	if len(stack) > 0 {
		sb.WriteString("\033[0m")
	}

	return sb.String()
}

func extractZip(zipBytes []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return err
	}
	cleanDest := filepath.Clean(destDir)
	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, cleanDest) {
			continue // Zip slip mitigation
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(cleanTarget, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		outFile, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024.0)
	} else {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024.0*1024.0))
	}
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func goValueToXruValue(v any) Value {
	if v == nil {
		return NilValue{}
	}
	switch val := v.(type) {
	case bool:
		return BoolValue{Val: val}
	case float64:
		return NumberValue{Val: val}
	case int:
		return NumberValue{Val: float64(val)}
	case string:
		return StringValue{Val: val}
	case []any:
		var elems []Value
		for _, item := range val {
			elems = append(elems, goValueToXruValue(item))
		}
		return ListValue{Elements: elems}
	case map[string]any:
		pairs := make(map[string]Value)
		for k, item := range val {
			pairs[k] = goValueToXruValue(item)
		}
		return MapValue{Pairs: pairs}
	default:
		return StringValue{Val: fmt.Sprintf("%v", val)}
	}
}

func xruValueToGoValue(v Value) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case NilValue:
		return nil
	case BoolValue:
		return val.Val
	case NumberValue:
		return val.Val
	case StringValue:
		return val.Val
	case DurationValue:
		return val.Val.String()
	case ListValue:
		var list []any
		for _, elem := range val.Elements {
			list = append(list, xruValueToGoValue(elem))
		}
		return list
	case MapValue:
		m := make(map[string]any)
		for k, elem := range val.Pairs {
			m[k] = xruValueToGoValue(elem)
		}
		return m
	default:
		return val.String()
	}
}

func parseTimeToTime(v Value) (time.Time, error) {
	if v == nil || v == (NilValue{}) {
		return time.Now(), nil
	}
	switch val := v.(type) {
	case NumberValue:
		n := int64(val.Val)
		if n > 100000000000 {
			return time.UnixMilli(n), nil
		}
		return time.Unix(n, 0), nil
	case StringValue:
		str := strings.TrimSpace(val.Val)
		if str == "" || strings.EqualFold(str, "now") {
			return time.Now(), nil
		}
		formats := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02",
			"02/01/2006",
			"02/01/2006 15:04:05",
			"15:04:05",
		}
		for _, f := range formats {
			if t, err := time.Parse(f, str); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("date: unable to parse date string '%s'", str)
	case MapValue:
		if isoVal, ok := val.Pairs["iso"]; ok {
			return time.Parse(time.RFC3339, isoVal.String())
		}
		if tsVal, ok := val.Pairs["timestamp"]; ok {
			if nVal, ok := tsVal.(NumberValue); ok {
				return time.UnixMilli(int64(nVal.Val)), nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("date: invalid date value of type %s", v.Type())
}

func dateToMap(t time.Time) MapValue {
	return MapValue{
		Pairs: map[string]Value{
			"iso":       StringValue{Val: t.Format(time.RFC3339)},
			"year":      NumberValue{Val: float64(t.Year())},
			"month":     NumberValue{Val: float64(t.Month())},
			"day":       NumberValue{Val: float64(t.Day())},
			"hour":      NumberValue{Val: float64(t.Hour())},
			"minute":    NumberValue{Val: float64(t.Minute())},
			"second":    NumberValue{Val: float64(t.Second())},
			"timestamp": NumberValue{Val: float64(t.UnixMilli())},
			"weekday":   StringValue{Val: t.Weekday().String()},
		},
	}
}

func convertDateFormat(layout string) string {
	r := layout
	r = strings.ReplaceAll(r, "YYYY", "2006")
	r = strings.ReplaceAll(r, "yyyy", "2006")
	r = strings.ReplaceAll(r, "YY", "06")
	r = strings.ReplaceAll(r, "MM", "01")
	r = strings.ReplaceAll(r, "DD", "02")
	r = strings.ReplaceAll(r, "dd", "02")
	r = strings.ReplaceAll(r, "HH", "15")
	r = strings.ReplaceAll(r, "hh", "03")
	r = strings.ReplaceAll(r, "mm", "04")
	r = strings.ReplaceAll(r, "ss", "05")
	r = strings.ReplaceAll(r, "SSS", "000")
	return r
}
