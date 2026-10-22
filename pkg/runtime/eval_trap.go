package runtime

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

type TrapHandler struct {
	Targets []string
	Alias   string
	Body    *ast.BlockStatement
	Env     *Environment
}

var (
	globalExitHooksLock  sync.Mutex
	globalExitHooks      []func(code int)
	globalSignalHandling atomic.Bool
	globalHasSignalTraps atomic.Bool
)

func IsSignalHandling() bool {
	return globalSignalHandling.Load()
}

func HasSignalTraps() bool {
	return globalHasSignalTraps.Load()
}

func RegisterGlobalExitHook(fn func(code int)) {
	globalExitHooksLock.Lock()
	defer globalExitHooksLock.Unlock()
	globalExitHooks = append(globalExitHooks, fn)
}

func TriggerExitHooks(code int) {
	globalExitHooksLock.Lock()
	hooks := globalExitHooks
	globalExitHooks = nil
	globalExitHooksLock.Unlock()

	for _, fn := range hooks {
		fn(code)
	}
}

func (ev *Evaluator) evalTrapStatement(s *ast.TrapStatement, env *Environment) (Value, error) {
	ev.trapMu.Lock()
	defer ev.trapMu.Unlock()

	hasSignal := false
	hasExit := false
	hasError := false

	for _, target := range s.Targets {
		switch strings.ToLower(target) {
		case "interrupt", "sigint", "terminate", "sigterm", "hangup", "sighup":
			hasSignal = true
		case "exit":
			hasExit = true
		case "error":
			hasError = true
		}
	}

	handler := &TrapHandler{
		Targets: s.Targets,
		Alias:   s.Alias,
		Body:    s.Body,
		Env:     env,
	}

	if hasSignal {
		globalHasSignalTraps.Store(true)
		ev.signalTraps = append(ev.signalTraps, handler)
		ev.startSignalListener()
	}
	if hasExit {
		ev.exitTraps = append(ev.exitTraps, handler)
		// Enregistrer aussi avec les hooks de sortie globale
		RegisterGlobalExitHook(func(code int) {
			ev.RunExitTraps(code)
		})
	}
	if hasError {
		ev.errorTraps = append(ev.errorTraps, handler)
	}

	return NilValue{}, nil
}

func (ev *Evaluator) startSignalListener() {
	if ev.sigListening {
		return
	}
	ev.sigListening = true
	ev.sigChan = make(chan os.Signal, 2)
	signal.Notify(ev.sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		for sig := range ev.sigChan {
			globalSignalHandling.Store(true)
			sigName := "interrupt"
			exitCode := 130
			if sig == syscall.SIGTERM {
				sigName = "terminate"
				exitCode = 143
			} else if sig == syscall.SIGHUP {
				sigName = "hangup"
				exitCode = 129
			}

			// Protection double Ctrl+C : deuxième signal = force kill immédiat
			go func() {
				secondSig := <-ev.sigChan
				if secondSig != nil {
					os.Exit(exitCode)
				}
			}()

			ev.trapMu.Lock()
			traps := make([]*TrapHandler, len(ev.signalTraps))
			copy(traps, ev.signalTraps)
			ev.trapMu.Unlock()

			matched := false
			for _, trap := range traps {
				for _, t := range trap.Targets {
					tLower := strings.ToLower(t)
					if tLower == sigName || (sigName == "interrupt" && tLower == "sigint") || (sigName == "terminate" && tLower == "sigterm") || (sigName == "hangup" && tLower == "sighup") {
						matched = true
						trapEnv := NewBlockEnvironment(trap.Env)
						if trap.Alias != "" {
							trapEnv.Set(trap.Alias, StringValue{Val: sigName}, false)
						}
						if _, trapErr := ev.Eval(trap.Body, trapEnv); trapErr != nil {
							fmt.Fprintf(os.Stderr, "\033[31m[TRAP ERROR]\033[0m %v\n", trapErr)
						}
						break
					}
				}
			}

			// Exécuter les exit traps avant d'arrêter le processus
			ev.RunExitTraps(exitCode)

			if matched {
				os.Exit(exitCode)
			} else {
				os.Exit(exitCode)
			}
		}
	}()
}

func (ev *Evaluator) RunExitTraps(code int) {
	ev.trapMu.Lock()
	traps := make([]*TrapHandler, len(ev.exitTraps))
	copy(traps, ev.exitTraps)
	ev.exitTraps = nil
	ev.trapMu.Unlock()

	for _, trap := range traps {
		trapEnv := NewBlockEnvironment(trap.Env)
		if trap.Alias != "" {
			trapEnv.Set(trap.Alias, NumberValue{Val: float64(code)}, false)
		}
		_, _ = ev.Eval(trap.Body, trapEnv)
	}
}

func (ev *Evaluator) RunErrorTraps(err error) bool {
	ev.trapMu.Lock()
	traps := make([]*TrapHandler, len(ev.errorTraps))
	copy(traps, ev.errorTraps)
	ev.trapMu.Unlock()

	if len(traps) == 0 {
		return false
	}
	for _, trap := range traps {
		trapEnv := NewBlockEnvironment(trap.Env)
		if trap.Alias != "" {
			trapEnv.Set(trap.Alias, StringValue{Val: err.Error()}, false)
		}
		_, _ = ev.Eval(trap.Body, trapEnv)
	}
	return true
}
