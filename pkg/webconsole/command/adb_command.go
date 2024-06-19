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

	"yunion.io/x/onecloud/pkg/mcclient"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

type SAdbShellInfo struct {
	HostIp   string `json:"host_ip"`
	HostPort int    `json:"host_port"`
}

type SAdbShellCommand struct {
	*BaseCommand
	Info *SAdbShellInfo
	s    *mcclient.ClientSession
}

func NewAdbShellCommand(info *SAdbShellInfo, s *mcclient.ClientSession) (*SAdbShellCommand, error) {
	name := o.Options.AdbPath
	var connStr string
	initCmd := exec.Command(name, "connect", fmt.Sprintf("%s:%d", info.HostIp, info.HostPort))
	if err := initCmd.Run(); err != nil {
		return nil, errors.Wrap(err, "connect adb")
	} else {
		connStr = fmt.Sprintf("%s:%d", info.HostIp, info.HostPort)
	}
	cmd := NewBaseCommand(s, name, "-s", connStr, "shell")
	tool := &SAdbShellCommand{
		BaseCommand: cmd,
		Info:        info,
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
	initCmd := exec.Command(o.Options.AdbPath, "disconnect", fmt.Sprintf("%s:%d", c.Info.HostIp, c.Info.HostPort))
	if err := initCmd.Run(); err != nil {
		return errors.Wrap(err, "disconnect adb")
	}
	return nil
}
