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

package command

import (
	"fmt"
	"os/exec"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

type SAdbShellInfo struct {
	IpAddr string `json:"ip_addr"`
	Port   int    `json:"port"`
}

type SAdbShellCommand struct {
	*BaseCommand
	Info *SAdbShellInfo
	s    *mcclient.ClientSession
}

func NewAdbShellCommand(info *SAdbShellInfo, s *mcclient.ClientSession) (*SAdbShellCommand, error) {
	if info.IpAddr == "" {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "Empty host ip address")
	}
	if info.Port == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "Empty host port")
	}
	name := o.Options.AdbPath
	cmd := NewBaseCommand(s, name, "-s", fmt.Sprintf("%s:%d", info.IpAddr, info.Port), "shell")
	tool := &SAdbShellCommand{
		BaseCommand: cmd,
		Info:        info,
	}
	initCmd := exec.Command(name, "connect", fmt.Sprintf("%s:%d", info.IpAddr, info.Port))
	if err := initCmd.Run(); err != nil {
		return nil, errors.Wrap(err, "connect adb")
	}
	return tool, nil
}

func (c *SAdbShellCommand) GetCommand() *exec.Cmd {
	return c.BaseCommand.GetCommand()
}

func (c SAdbShellCommand) GetProtocol() string {
	return PROTOCOL_TTY
}

func (c *SAdbShellCommand) Cleanup() error {
	initCmd := exec.Command(o.Options.AdbPath, "disconnect", fmt.Sprintf("%s:%d", c.Info.IpAddr, c.Info.Port))
	if err := initCmd.Run(); err != nil {
		return errors.Wrap(err, "disconnect adb")
	}
	return nil
}
