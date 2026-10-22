package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"
	"time"
)

func newCmdUnit() *UnitInstance {
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
				exitCode := exitErr.ExitCode()
				if exitCode == -1 || exitCode == 130 || exitCode == 143 || IsSignalHandling() {
					// Si un signal est en cours de traitement, laisser la priorité absolue au trap goroutine
					for i := 0; i < 30; i++ {
						if IsSignalHandling() {
							select {} // Bloque le thread principal : le trap effectuera le cleanup et appellera os.Exit
						}
						time.Sleep(10 * time.Millisecond)
					}
				}
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
	return cmdUnit
}
