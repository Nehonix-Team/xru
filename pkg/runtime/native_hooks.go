package runtime

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
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

// NewNativeHooks returns the map of low-level kernel and engine hooks
// that can be invoked via @native("...") functions in XRU modules.
func NewNativeHooks() map[string]NativeMethod {
	hooks := make(map[string]NativeMethod)

	// Crypto
	hooks["crypto_sha256"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return StringValue{Val: ""}, nil
		}
		data := args[0].String()
		h := sha256.Sum256([]byte(data))
		return StringValue{Val: hex.EncodeToString(h[:])}, nil
	}

	hooks["crypto_file_sha256"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, fmt.Errorf("crypto_file_sha256 requires file path")
		}
		target := args[0].String()
		h, err := hashFile(target, "sha256")
		if err != nil {
			return NilValue{}, err
		}
		return StringValue{Val: h}, nil
	}

	hooks["crypto_verify_file"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("crypto_verify_file requires file path and expected hash")
		}
		target := args[0].String()
		expected := strings.TrimSpace(strings.ToLower(args[1].String()))
		algo := "sha256"
		if len(args) >= 3 {
			algo = strings.TrimSpace(args[2].String())
		}
		actual, err := hashFile(target, algo)
		if err != nil {
			return BoolValue{Val: false}, err
		}
		match := subtle.ConstantTimeCompare([]byte(strings.ToLower(actual)), []byte(expected)) == 1
		return BoolValue{Val: match}, nil
	}

	hooks["crypto_md5"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return StringValue{Val: ""}, nil
		}
		data := args[0].String()
		h := md5.Sum([]byte(data))
		return StringValue{Val: hex.EncodeToString(h[:])}, nil
	}

	hooks["crypto_random_bytes"] = func(args []Value) (Value, error) {
		length := 32
		if len(args) > 0 {
			if n, ok := args[0].(NumberValue); ok {
				length = int(n.Val)
			}
		}
		buf := make([]byte, length)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		return StringValue{Val: hex.EncodeToString(buf)}, nil
	}

	// System & Process
	hooks["sys_exit"] = func(args []Value) (Value, error) {
		code := 0
		if len(args) > 0 {
			if n, ok := args[0].(NumberValue); ok {
				code = int(n.Val)
			}
		}
		os.Exit(code)
		return NilValue{}, nil
	}

	hooks["sys_os"] = func(args []Value) (Value, error) {
		return StringValue{Val: goRuntime.GOOS}, nil
	}

	hooks["sys_arch"] = func(args []Value) (Value, error) {
		return StringValue{Val: goRuntime.GOARCH}, nil
	}

	hooks["sys_getenv"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return StringValue{Val: ""}, nil
		}
		key := args[0].String()
		val := os.Getenv(key)
		if val == "" && len(args) > 1 {
			val = args[1].String()
		}
		return StringValue{Val: val}, nil
	}

	hooks["sys_time"] = func(args []Value) (Value, error) {
		return StringValue{Val: time.Now().Format(time.RFC3339)}, nil
	}

	hooks["sys_cwd"] = func(args []Value) (Value, error) {
		wd, err := os.Getwd()
		if err != nil {
			return StringValue{Val: ""}, err
		}
		if abs, err := filepath.Abs(wd); err == nil {
			return StringValue{Val: abs}, nil
		}
		return StringValue{Val: wd}, nil
	}

	hooks["sys_sleep"] = func(args []Value) (Value, error) {
		if len(args) > 0 {
			if dur, ok := args[0].(DurationValue); ok {
				time.Sleep(dur.Val)
				return BoolValue{Val: true}, nil
			}
			if num, ok := args[0].(NumberValue); ok {
				time.Sleep(time.Duration(num.Val) * time.Millisecond)
				return BoolValue{Val: true}, nil
			}
		}
		return BoolValue{Val: false}, nil
	}

	// Console I/O
	hooks["console_println"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			fmt.Println()
			return NilValue{}, nil
		}
		fmt.Println(formatTerminalColors(args[0].String()))
		return NilValue{}, nil
	}

	hooks["console_print"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		fmt.Print(formatTerminalColors(args[0].String()))
		return NilValue{}, nil
	}

	// File System
	hooks["fs_exists"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return BoolValue{Val: false}, nil
		}
		_, err := os.Stat(args[0].String())
		return BoolValue{Val: err == nil}, nil
	}

	hooks["fs_is_dir"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return BoolValue{Val: false}, nil
		}
		fi, err := os.Stat(args[0].String())
		return BoolValue{Val: err == nil && fi.IsDir()}, nil
	}

	hooks["fs_is_file"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return BoolValue{Val: false}, nil
		}
		fi, err := os.Stat(args[0].String())
		return BoolValue{Val: err == nil && !fi.IsDir()}, nil
	}

	hooks["fs_read"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return StringValue{Val: ""}, nil
		}
		content, err := os.ReadFile(args[0].String())
		if err != nil {
			return nil, err
		}
		return StringValue{Val: string(content)}, nil
	}

	hooks["fs_write"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return BoolValue{Val: false}, fmt.Errorf("fs_write requires path and content")
		}
		target := args[0].String()
		content := args[1].String()
		_ = os.MkdirAll(filepath.Dir(target), 0755)
		err := os.WriteFile(target, []byte(content), 0644)
		return BoolValue{Val: err == nil}, err
	}

	hooks["fs_remove"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		err := os.RemoveAll(args[0].String())
		return NilValue{}, err
	}

	hooks["fs_move"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("fs_move requires source and destination")
		}
		err := os.Rename(args[0].String(), args[1].String())
		return NilValue{}, err
	}

	hooks["fs_ensure_dir"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		err := os.MkdirAll(args[0].String(), 0755)
		return BoolValue{Val: err == nil}, err
	}

	hooks["fs_size"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NumberValue{Val: 0}, nil
		}
		fi, err := os.Stat(args[0].String())
		if err != nil {
			return NumberValue{Val: 0}, err
		}
		return NumberValue{Val: float64(fi.Size())}, nil
	}

	// Command execution
	hooks["cmd_exec"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return StringValue{Val: ""}, nil
		}
		cmdStr := args[0].String()
		var cmd *exec.Cmd
		if goRuntime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", cmdStr)
		} else {
			cmd = exec.Command("sh", "-c", cmdStr)
		}
		out, err := cmd.Output()
		if err != nil {
			return StringValue{Val: string(out)}, err
		}
		return StringValue{Val: string(out)}, nil
	}

	// Date & Time
	hooks["date_now"] = func(args []Value) (Value, error) {
		return dateToMap(time.Now()), nil
	}

	hooks["date_iso"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return StringValue{Val: time.Now().Format(time.RFC3339)}, nil
		}
		t, err := parseTimeToTime(args[0])
		if err != nil {
			return NilValue{}, err
		}
		return StringValue{Val: t.Format(time.RFC3339)}, nil
	}

	hooks["date_timestamp"] = func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NumberValue{Val: float64(time.Now().UnixNano() / int64(time.Millisecond))}, nil
		}
		t, err := parseTimeToTime(args[0])
		if err != nil {
			return NilValue{}, err
		}
		return NumberValue{Val: float64(t.UnixNano() / int64(time.Millisecond))}, nil
	}

	hooks["date_parse"] = func(args []Value) (Value, error) {
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
	}

	hooks["date_add"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("date.add requires at least 2 arguments (date, amount, [unit])")
		}
		t, err := parseTimeToTime(args[0])
		if err != nil {
			return NilValue{}, err
		}
		if len(args) == 2 {
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
	}

	hooks["date_format"] = func(args []Value) (Value, error) {
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
	}

	// JSON
	hooks["json_parse"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("json.parse requires raw string")
		}
		raw := args[0].String()
		var parsed any
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return NilValue{}, fmt.Errorf("json.parse failed: %w", err)
		}
		return goValueToXruValue(parsed), nil
	}

	hooks["json_stringify"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("json.stringify requires 1 argument")
		}
		goVal := xruValueToGoValue(args[0])
		var b []byte
		var err error
		if len(args) >= 2 && args[1].String() != "" {
			b, err = json.MarshalIndent(goVal, "", args[1].String())
		} else {
			b, err = json.Marshal(goVal)
		}
		if err != nil {
			return NilValue{}, fmt.Errorf("json.stringify failed: %w", err)
		}
		return StringValue{Val: string(b)}, nil
	}

	// Net
	hooks["net_get"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("net.get requires url")
		}
		url := args[0].String()
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

	hooks["net_fetch"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("net.fetch requires url")
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

	hooks["net_clone"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("net.clone requires repo url and dest")
		}
		repo := args[0].String()
		dest := args[1].String()

		cmd := exec.Command("git", "clone", "--depth", "1", repo, dest)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		return BoolValue{Val: err == nil}, err
	}

	return hooks
}
