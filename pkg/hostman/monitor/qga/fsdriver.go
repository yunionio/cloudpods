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

package qga

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils/driver"
)

type SQgaDriver struct {
	agent *QemuGuestAgent
}

func NewQgaFsutilDriver(agent *QemuGuestAgent) driver.IFsutilExecDriver {
	ret := new(SQgaDriver)
	ret.agent = agent
	return ret
}

func (q *SQgaDriver) ExecInputWait(name string, args []string, input []string) (int, string, string, error) {
	return q.agent.CommandWithTimeout(name, args, nil, "", true, -1)
}

func (q *SQgaDriver) Exec(name string, args ...string) ([]byte, error) {
	retCode, stdout, stderr, err := q.agent.CommandWithTimeout(name, args, nil, "", true, -1)
	if err != nil {
		return nil, err
	}
	if retCode != 0 {
		return []byte(stdout + "\n" + stderr), errors.Errorf("Exit code %d", retCode)
	}
	var retStr = []byte(stdout)
	if len(stderr) > 0 {
		retStr = []byte(stdout + "\n" + stderr)
	}
	return retStr, nil
}

func (q *SQgaDriver) Run(name string, args ...string) error {
	retCode, stdout, stderr, err := q.agent.CommandWithTimeout(name, args, nil, "", true, -1)
	if err != nil {
		return err
	}
	if retCode != 0 {
		return errors.Errorf("Exit code %d\n%s\n%s", retCode, stdout, stderr)
	}
	return nil
}
