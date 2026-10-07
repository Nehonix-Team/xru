package runtime

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"time"
)

func NewBuiltinUnits() map[string]*UnitInstance {
	units := make(map[string]*UnitInstance)

	// 1. #UNIT Utils
	utilsUnit := &UnitInstance{
		UnitName:   "Utils",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Log": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return NilValue{}, nil
				}
				msg := formatTerminalColors(args[0].String())
				fmt.Println(msg)
				return NilValue{}, nil
			},
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
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Utils"] = utilsUnit

	// 2. #UNIT Fs
	fsUnit := &UnitInstance{
		UnitName:   "Fs",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"EnsureDir": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Fs.EnsureDir requires directory path")
				}
				err := os.MkdirAll(args[0].String(), 0755)
				return BoolValue{Val: err == nil}, err
			},
			"Remove": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Fs.Remove requires target path")
				}
				err := os.RemoveAll(args[0].String())
				return BoolValue{Val: err == nil}, err
			},
			"Move": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("Fs.Move requires src and dest path")
				}
				err := os.Rename(args[0].String(), args[1].String())
				return BoolValue{Val: err == nil}, err
			},
			"Exists": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return BoolValue{Val: false}, nil
				}
				_, err := os.Stat(args[0].String())
				return BoolValue{Val: !os.IsNotExist(err)}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Fs"] = fsUnit

	// 3. #UNIT Net
	netUnit := &UnitInstance{
		UnitName:   "Net",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Fetch": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("Net.Fetch requires url and dest path")
				}
				url := args[0].String()
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
			},
			"Clone": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("Net.Clone requires repo url and dest")
				}
				repo := args[0].String()
				dest := args[1].String()

				cmd := exec.Command("git", "clone", "--depth", "1", repo, dest)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				return BoolValue{Val: err == nil}, err
			},
		},
		Directives: map[string]NativeDirective{
			"#SET": func(key Value, val Value) error {
				return nil
			},
			"#HEADER": func(key Value, val Value) error {
				return nil
			},
			"#ON": func(key Value, val Value) error {
				return nil
			},
		},
	}
	units["Net"] = netUnit

	// 4. #UNIT Sys
	sysUnit := &UnitInstance{
		UnitName:   "Sys",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Env": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return StringValue{Val: ""}, nil
				}
				return StringValue{Val: os.Getenv(args[0].String())}, nil
			},
			"Time": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format(time.RFC3339)}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Sys"] = sysUnit

	// 5. #UNIT Cmd (exécution de commandes terminal cross-platform)
	cmdUnit := &UnitInstance{
		UnitName:   "Cmd",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Exec": func(args []Value) (Value, error) {
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
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				return BoolValue{Val: err == nil}, err
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Cmd"] = cmdUnit

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

func formatTerminalColors(s string) string {
	s = strings.ReplaceAll(s, "<green>", "\033[32m")
	s = strings.ReplaceAll(s, "<yellow>", "\033[33m")
	s = strings.ReplaceAll(s, "<cyan>", "\033[36m")
	s = strings.ReplaceAll(s, "<blue>", "\033[34m")
	s = strings.ReplaceAll(s, "<red>", "\033[31m")
	s = strings.ReplaceAll(s, "<bold>", "\033[1m")
	s = strings.ReplaceAll(s, "</>", "\033[0m")
	return s
}
