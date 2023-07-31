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

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SocatInfo struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type SocatCmd struct {
	*BaseCommand
	Info *SocatInfo
	s    *mcclient.ClientSession
}

func NewSocatCommand(info *SocatInfo, s *mcclient.ClientSession) (*SocatCmd, error) {
	if info.IP == "" {
		return nil, fmt.Errorf("Empty remote ip address")
	}
	if info.Port <= 0 {
		return nil, fmt.Errorf("Invalid port %d", info.Port)
	}

	name := "bash"
	args := []string{
		"-c",
		fmt.Sprintf("socat FILE:`tty`,raw,echo=0 TCP:%s:%d", info.IP, info.Port),
	}
	cmd := NewBaseCommand(s, name, args...)
	scmd := &SocatCmd{
		BaseCommand: cmd,
		Info:        info,
	}

	return scmd, nil
}

func (c *SocatCmd) GetCommand() *exec.Cmd {
	return c.BaseCommand.GetCommand()
}

func (c *SocatCmd) GetProtocol() string {
	return PROTOCOL_TTY
}
