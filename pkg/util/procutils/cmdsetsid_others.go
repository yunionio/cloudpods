// +build !windows

package procutils

import (
	"os/exec"
	"syscall"
)

func cmdSetSid(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
