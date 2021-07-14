// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
