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
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (ev *Evaluator) evalCoutStatement(s *ast.CoutStatement, env *Environment) (Value, error) {
	defaultDir := ""
	captureOut := true
	captureErr := true

	for _, opt := range s.Options {
		key := strings.ToLower(opt.Key)
		if key == "dir" || key == "cwd" {
			if opt.Value != nil {
				val, err := ev.Eval(opt.Value, env)
				if err != nil {
					return nil, err
				}
				defaultDir = val.String()
			}
		} else if key == "out" {
			captureOut = true
		} else if key == "err" {
			captureErr = true
		}
	}

	blockEnv := NewBlockEnvironment(env)

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	lastExitCode := 0
	allOk := true

	prevExec, hadExec := env.GetGlobal("exec")

	customExecFn := func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("::exec requires command string")
		}
		commandLine := args[0].String()
		var cmd *exec.Cmd
		if goRuntime.GOOS == "windows" {
			cmd = exec.Command("cmd.exe", "/c", commandLine)
		} else {
			cmd = exec.Command("sh", "-c", commandLine)
		}

		workDir := defaultDir
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

		var cmdOutBuf, cmdErrBuf bytes.Buffer
		if captureOut {
			cmd.Stdout = io.MultiWriter(&stdoutBuf, &cmdOutBuf)
		} else {
			cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf, &cmdOutBuf)
		}

		if captureErr {
			cmd.Stderr = io.MultiWriter(&stderrBuf, &cmdErrBuf)
		} else {
			cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf, &cmdErrBuf)
		}

		runErr := cmd.Run()
		exitCode := 0
		cmdOk := true
		if runErr != nil {
			var exitErr *exec.ExitError
			if errors.As(runErr, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
			lastExitCode = exitCode
			allOk = false
			cmdOk = false
		}

		return MapValue{
			Pairs: map[string]Value{
				"out":  StringValue{Val: strings.TrimRight(cmdOutBuf.String(), "\r\n")},
				"err":  StringValue{Val: strings.TrimRight(cmdErrBuf.String(), "\r\n")},
				"code": NumberValue{Val: float64(exitCode)},
				"ok":   BoolValue{Val: cmdOk},
			},
			InterfaceName: "CmdResult",
		}, nil
	}

	env.SetGlobal("exec", FunctionValue{Fn: customExecFn})

	defer func() {
		if hadExec {
			env.SetGlobal("exec", prevExec)
		} else {
			delete(env.globals, "exec")
		}
	}()

	if s.Body != nil {
		_, err := ev.Eval(s.Body, blockEnv)
		if err != nil {
			return nil, err
		}
	}

	resultMap := MapValue{
		Pairs: map[string]Value{
			"out":  StringValue{Val: strings.TrimRight(stdoutBuf.String(), "\r\n")},
			"err":  StringValue{Val: strings.TrimRight(stderrBuf.String(), "\r\n")},
			"code": NumberValue{Val: float64(lastExitCode)},
			"ok":   BoolValue{Val: allOk},
		},
		InterfaceName: "CmdResult",
	}

	if s.Alias != "" {
		_ = env.Set(s.Alias, resultMap, false)
	}

	return resultMap, nil
}
