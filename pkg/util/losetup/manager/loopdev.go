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

package manager

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/losetup"
)

var managerObj iManager

func init() {
	managerObj = newManager()
}

func ListDevices() (*losetup.Devices, error) {
	return managerObj.listDevices()
}

func AttachDevice(filePath string, partScan bool) (*losetup.Device, error) {
	return managerObj.attachDevice(filePath, partScan)
}

func DetachDevice(filePath string) error {
	return managerObj.detachDevice(filePath)
}

type iManager interface {
	listDevices() (*losetup.Devices, error)
	attachDevice(filePath string, partScan bool) (*losetup.Device, error)
	detachDevice(devPath string) error
}

type execType string

const (
	EXEC_ATTACH execType = "attach"
	EXEC_DETACH execType = "detach"
)

type execAttach struct {
	FilePath string
	PartScan bool
}

type execDetach struct {
	DevicePath string
}

type execOption struct {
	Type   execType
	Attach *execAttach
	Detach *execDetach
	result chan execResult
}

type execResult struct {
	error          error
	attachedDevice *losetup.Device
}

func newAttachOption(filePath string, partScan bool) *execOption {
	return &execOption{
		Type:   EXEC_ATTACH,
		result: make(chan execResult),
		Attach: &execAttach{
			FilePath: filePath,
			PartScan: partScan,
		},
	}
}

func newDetachOption(devPath string) *execOption {
	return &execOption{
		Type:   EXEC_DETACH,
		result: make(chan execResult),
		Detach: &execDetach{
			DevicePath: devPath,
		},
	}
}

type manager struct {
	execOptCh chan *execOption
}

func newManager() iManager {
	m := &manager{
		execOptCh: make(chan *execOption),
	}
	go func() {
		m.startExec()
	}()
	return m
}

func (m *manager) listDevices() (*losetup.Devices, error) {
	return losetup.ListDevices()
}

func (m *manager) attachDevice(filePath string, partScan bool) (*losetup.Device, error) {
	opt := newAttachOption(filePath, partScan)
	m.execOptCh <- opt
	result := <-opt.result
	return result.attachedDevice, result.error
}

func (m *manager) detachDevice(devPath string) error {
	opt := newDetachOption(devPath)
	m.execOptCh <- opt
	result := <-opt.result
	return result.error
}

func (m *manager) startExec() {
	for execOpt := range m.execOptCh {
		switch execOpt.Type {
		case EXEC_ATTACH:
			log.Infof("attach %s", jsonutils.Marshal(execOpt.Attach))
			dev, err := losetup.AttachDevice(execOpt.Attach.FilePath, execOpt.Attach.PartScan)
			execOpt.result <- execResult{
				attachedDevice: dev,
				error:          err,
			}
		case EXEC_DETACH:
			log.Infof("detach %s", jsonutils.Marshal(execOpt.Detach))
			err := losetup.DetachDevice(execOpt.Detach.DevicePath)
			execOpt.result <- execResult{
				error: err,
			}
		default:
			err := errors.Errorf("unknown exec type: %s", execOpt.Type)
			execOpt.result <- execResult{
				error: err,
			}
		}
	}
}
