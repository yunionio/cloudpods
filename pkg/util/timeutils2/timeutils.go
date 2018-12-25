package timeutils2

import (
	"fmt"
	"os/exec"
	"time"
)

func AddTimeout(second time.Duration, callback func()) {
	go func() {
		<-time.NewTimer(second).C
		callback()
	}()
}

func CommandWithTimeout(timeout int, cmds ...string) *exec.Cmd {
	if timeout > 0 {
		cmds = append([]string{"timeout", "--signal=KILL", fmt.Sprintf("%ds", timeout)}, cmds...)
	}
	return exec.Command(cmds[0], cmds[1:]...)
}
