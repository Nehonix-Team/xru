package runtime

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math"
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

	stdUnit := &UnitInstance{
		UnitName:   "std",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"println":        printlnFn,
			"Println":        printlnFn,
			"print":          printFn,
			"Print":          printFn,
			"info":           infoFn,
			"Info":           infoFn,
			"success":        successFn,
			"Success":        successFn,
			"warn":           warnFn,
			"Warn":           warnFn,
			"error":          errorFn,
			"Error":          errorFn,
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

				// Contexte étendu pour bloc #WITH Net.Fetch("...")
				var onProgress LambdaValue
				var onComplete LambdaValue
				props := make(map[string]Value)
				headers := make(map[string]string)
				props["url"] = StringValue{Val: url}

				fetchCtx := &UnitInstance{
					UnitName:   "Net.Fetch",
					Properties: props,
					Directives: map[string]NativeDirective{
						"set": func(key Value, val Value) error {
							k := key.String()
							switch strings.ToLower(k) {
							case "method", "timeout", "extractto", "dest":
								props[k] = val
								return nil
							default:
								return fmt.Errorf("unknown property '%s' on Net.fetch (supported: method, timeout, extractTo, dest)", k)
							}
						},
						"header": func(key Value, val Value) error {
							headers[key.String()] = val.String()
							return nil
						},
						"on": func(key Value, val Value) error {
							kStr := strings.ToLower(key.String())
							if kStr == "progress" {
								if lambda, ok := val.(LambdaValue); ok {
									onProgress = lambda
								}
								return nil
							} else if kStr == "complete" {
								if lambda, ok := val.(LambdaValue); ok {
									onComplete = lambda
								}
								return nil
							}
							return fmt.Errorf("unknown event '%s' on Net.fetch (supported: progress, complete)", key.String())
						},
					},
					Methods: map[string]NativeMethod{
						"__complete": func(mArgs []Value) (Value, error) {
							ev := NewEvaluator()

							// 1. Détermination de la méthode HTTP
							method := "GET"
							if m, ok := props["Method"]; ok && m.String() != "" {
								method = strings.ToUpper(m.String())
							}

							// 2. Détermination du timeout
							timeout := 30 * time.Second
							if tVal, ok := props["Timeout"]; ok {
								if dur, ok := tVal.(DurationValue); ok {
									timeout = dur.Val
								} else if num, ok := tVal.(NumberValue); ok {
									timeout = time.Duration(num.Val) * time.Second
								} else if parsed, err := time.ParseDuration(tVal.String()); err == nil {
									timeout = parsed
								}
							}

							// 3. Corps de requête optionnel
							var reqBody io.Reader
							if bVal, ok := props["Body"]; ok && bVal.String() != "" {
								reqBody = strings.NewReader(bVal.String())
							}

							extractTo := ""
							if et, ok := props["ExtractTo"]; ok {
								extractTo = et.String()
							}
							dest := ""
							if d, ok := props["Dest"]; ok {
								dest = d.String()
							}

							// 4. Initialisation de la requête HTTP
							req, err := http.NewRequest(method, url, reqBody)
							if err != nil {
								if onComplete.Body != nil {
									_, _ = ev.CallLambda(onComplete, []Value{NilValue{}, StringValue{Val: err.Error()}})
								}
								return NilValue{}, err
							}

							// En-têtes HTTP configurés via #HEADER
							for k, v := range headers {
								req.Header.Set(k, v)
							}

							client := &http.Client{Timeout: timeout}
							resp, err := client.Do(req)
							if err != nil {
								if onComplete.Body != nil {
									_, _ = ev.CallLambda(onComplete, []Value{NilValue{}, StringValue{Val: err.Error()}})
								}
								return NilValue{}, err
							}
							defer resp.Body.Close()

							// 5. Téléchargement réel et calcul de progression
							var bodyBuf bytes.Buffer
							buf := make([]byte, 32*1024)
							totalSize := resp.ContentLength
							var downloaded int64 = 0
							startTime := time.Now()
							lastProgress := time.Now()

							for {
								n, rErr := resp.Body.Read(buf)
								if n > 0 {
									downloaded += int64(n)
									bodyBuf.Write(buf[:n])

									if onProgress.Body != nil && time.Since(lastProgress) >= 150*time.Millisecond {
										lastProgress = time.Now()
										pct := 100
										if totalSize > 0 {
											pct = int((float64(downloaded) / float64(totalSize)) * 100)
										}
										elapsed := time.Since(startTime).Seconds()
										speed := formatSpeed(float64(downloaded) / math.Max(elapsed, 0.001))
										_, _ = ev.CallLambda(onProgress, []Value{
											StringValue{Val: fmt.Sprintf("%d", pct)},
											StringValue{Val: speed},
										})
									}
								}
								if rErr != nil {
									if rErr != io.EOF {
										if onComplete.Body != nil {
											_, _ = ev.CallLambda(onComplete, []Value{NilValue{}, StringValue{Val: rErr.Error()}})
										}
										return NilValue{}, rErr
									}
									break
								}
							}

							// Déclenchement de la notification de progression finale (100%)
							if onProgress.Body != nil {
								elapsed := time.Since(startTime).Seconds()
								speed := formatSpeed(float64(downloaded) / math.Max(elapsed, 0.001))
								_, _ = ev.CallLambda(onProgress, []Value{
									StringValue{Val: "100"},
									StringValue{Val: speed},
								})
							}

							// 6. Extraction ZIP ou écriture dans le fichier de destination
							dataBytes := bodyBuf.Bytes()
							if extractTo != "" {
								// Signature d'archive ZIP : PK\x03\x04 ou PK\x05\x06
								if len(dataBytes) >= 4 && dataBytes[0] == 0x50 && dataBytes[1] == 0x4B {
									if err := extractZip(dataBytes, extractTo); err != nil {
										if onComplete.Body != nil {
											_, _ = ev.CallLambda(onComplete, []Value{NilValue{}, StringValue{Val: fmt.Sprintf("zip extraction failed: %v", err)}})
										}
										return NilValue{}, err
									}
								}
							}
							if dest != "" {
								if err := os.MkdirAll(filepath.Dir(dest), 0755); err == nil {
									_ = os.WriteFile(dest, dataBytes, 0644)
								}
							}

							// 7. Déclenchement de l'événement Complete(data, err)
							var dataVal Value = StringValue{Val: bodyBuf.String()}
							var errVal Value = NilValue{}
							if resp.StatusCode >= 400 {
								errVal = StringValue{Val: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)}
							}

							if onComplete.Body != nil {
								_, _ = ev.CallLambda(onComplete, []Value{dataVal, errVal})
							}

							return dataVal, nil
						},
					},
				}
				return fetchCtx, nil
			}
	cloneFn := func(args []Value) (Value, error) {
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
	}

	netUnit := &UnitInstance{
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
