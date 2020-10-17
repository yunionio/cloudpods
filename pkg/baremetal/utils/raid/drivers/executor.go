package drivers

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/ssh"
)

type Executor struct{}

func (e *Executor) Run(cmds ...string) ([]string, error) {
	return e.run(true, cmds, nil)
}

func (e *Executor) RunWithInput(input io.Reader, cmds ...string) ([]string, error) {
	return e.run(true, cmds, input)
}

func (e *Executor) run(parseOutput bool, cmds []string, input io.Reader) ([]string, error) {
	ret := []string{}
	for _, cmd := range cmds {
		log.Debugf("Run command: %s", cmd)
		proc := exec.Command("sh", "-c", cmd)
		var stdOut bytes.Buffer
		var stdErr bytes.Buffer
		proc.Stdout = &stdOut
		proc.Stderr = &stdErr
		proc.Stdin = input
		if err := proc.Run(); err != nil {
			var outputErr error
			errMsg := stdErr.String()
			if len(stdOut.String()) != 0 {
				errMsg = fmt.Sprintf("%s %s", errMsg, stdOut.String())
			}
			outputErr = errors.Error(errMsg)
			err = errors.Wrapf(outputErr, "%q error: %v, cmd error", cmd, err)
			return nil, err
		}
		if parseOutput {
			ret = append(ret, ssh.ParseOutput(stdOut.Bytes())...)
		} else {
			ret = append(ret, stdOut.String())
		}
	}
	return ret, nil
}

func NewExecutor() *Executor {
	return new(Executor)
}
