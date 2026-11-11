//go:build !windows

package runtime

import "os/exec"

func createOsCmd(commandLine string) *exec.Cmd {
	return exec.Command("sh", "-c", commandLine)
}
