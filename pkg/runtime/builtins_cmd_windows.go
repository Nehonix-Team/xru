//go:build windows

package runtime

import (
	"fmt"
	"os/exec"
	"syscall"
)

func createOsCmd(commandLine string) *exec.Cmd {
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: fmt.Sprintf(`cmd.exe /s /c "%s"`, commandLine),
	}
	return cmd
}
