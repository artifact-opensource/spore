//go:build windows

package tools

import (
	"os/exec"
)

func setProcGroup(cmd *exec.Cmd) {
	// Windows doesn't support Setpgid; process groups handled differently
}

func killProcGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}
