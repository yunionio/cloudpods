package timeutils2

import (
	"fmt"
	"os/exec"
	"runtime/debug"
	"time"

	"yunion.io/x/log"
)

func AddTimeout(second time.Duration, callback func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorln(r)
				debug.PrintStack()
			}
		}()

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
