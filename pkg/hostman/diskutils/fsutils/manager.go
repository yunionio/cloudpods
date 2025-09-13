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

package fsutils

import "yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils/driver"

type SFsutilDriver struct {
	execDriver driver.IFsutilExecDriver
}

func NewFsutilDriver(execDriver driver.IFsutilExecDriver) *SFsutilDriver {
	return &SFsutilDriver{execDriver}
}

func (d *SFsutilDriver) Exec(name string, args ...string) ([]byte, error) {
	return d.execDriver.Exec(name, args...)
}

func (d *SFsutilDriver) Run(name string, args ...string) error {
	return d.execDriver.Run(name, args...)
}

func (d *SFsutilDriver) ExecInputWait(name string, args []string, input []string) (int, string, string, error) {
	return d.execDriver.ExecInputWait(name, args, input)
}
