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

package driver

import (
	"fmt"
	"io"
	"io/ioutil"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

type IFsutilExecDriver interface {
	Run(name string, args ...string) error
	Exec(name string, args ...string) ([]byte, error)
	ExecInputWait(name string, args []string, input []string) (int, string, string, error)
}

type SProcDriver struct {
}

func NewProcDriver() IFsutilExecDriver {
	return new(SProcDriver)
}

func (*SProcDriver) Exec(name string, args ...string) ([]byte, error) {
	return procutils.NewCommand(name, args...).Output()
}

func (*SProcDriver) Run(name string, args ...string) error {
	return procutils.NewCommand(name, args...).Run()
}

func (*SProcDriver) ExecInputWait(name string, args []string, input []string) (int, string, string, error) {
	proc := procutils.NewCommand(name, args...)
	stdin, err := proc.StdinPipe()
	if err != nil {
		return -1, "", "", err
	}
	defer stdin.Close()

	outb, err := proc.StdoutPipe()
	if err != nil {
		return -1, "", "", err
	}
	defer outb.Close()

	errb, err := proc.StderrPipe()
	if err != nil {
		return -1, "", "", err
	}
	defer errb.Close()
	if err := proc.Start(); err != nil {
		return -1, "", "", err
	}
	for _, s := range input {
		io.WriteString(stdin, fmt.Sprintf("%s\n", s))
	}
	stdoutPut, err := ioutil.ReadAll(outb)
	if err != nil {
		return -1, "", "", err
	}
	stderrOutPut, err := ioutil.ReadAll(errb)
	if err != nil {
		return -1, "", "", err
	}
	if err = proc.Wait(); err != nil {
		if status, succ := proc.GetExitStatus(err); succ {
			return status, string(stdoutPut), string(stderrOutPut), err
		} else {
			return 0, string(stdoutPut), string(stderrOutPut), err
		}
	}
	return 0, string(stdoutPut), string(stderrOutPut), nil
}
