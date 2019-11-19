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

package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"

	"yunion.io/x/pkg/errors"
)

const errPidNotRunning = errors.Error("pid not running")

type PidFile struct {
	Path string
	Comm string
}

func NewPidFile(path, comm string) *PidFile {
	pf := &PidFile{
		Path: path,
		Comm: comm,
	}
	return pf
}

func (pf *PidFile) findProcess() (*os.Process, error) {
	data, err := ioutil.ReadFile(pf.Path)
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, errPidNotRunning
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return nil, errPidNotRunning
	}
	return proc, nil
}

func (pf *PidFile) findComm(pid int) (string, error) {
	fp := fmt.Sprintf("/proc/%d/comm", pid)
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return "", err
	}
	comm := strings.TrimSpace(string(data))
	return comm, nil
}

// ConfirmedOrUnlink reads the pidfile, confirm that it's running and comm
// matches pf.Comm
//
// When confirmed is true, proc must be non-nil
func (pf *PidFile) ConfirmOrUnlink() (proc *os.Process, confirmed bool, err error) {
	proc, err = pf.findProcess()
	if err != nil {
		if err == errPidNotRunning {
			os.Remove(pf.Path)
		}
		err = errors.Wrapf(err, "pidfile %s: find process: %v", pf.Path, err)
		return
	}
	comm, err := pf.findComm(proc.Pid)
	if err != nil {
		err = errors.Wrapf(err, "pidfile %s: find comm: %v", pf.Path, err)
		return
	}
	if pf.Comm == comm {
		confirmed = true
		return
	}
	err = os.Remove(pf.Path)
	if err != nil {
		err = errors.Wrapf(err, "pidfile %s: unlink: %v", pf.Path, err)
	}
	return
}

func WritePidFile(pid int, pidFile string) error {
	data := fmt.Sprintf("%d\n", pid)
	err := ioutil.WriteFile(pidFile, []byte(data), FileModeFile)
	return err
}
