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

	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

type IpmiInfo struct {
	IpAddr   string `json:"ip_addr"`
	Username string `json:"username"`
	Password string `json:"password"`
	Present  bool   `json:"present"`
}

type IpmitoolSol struct {
	*BaseCommand
	Info *IpmiInfo
}

func NewIpmitoolSolCommand(info *IpmiInfo) (*IpmitoolSol, error) {
	if info.IpAddr == "" {
		return nil, fmt.Errorf("Empty host ip address")
	}
	if info.Username == "" {
		return nil, fmt.Errorf("Empty username")
	}
	if info.Password == "" {
		return nil, fmt.Errorf("Empty password")
	}
	name := o.Options.IpmitoolPath
	cmd := NewBaseCommand(name, "-I", "lanplus")
	cmd.AppendArgs("-H", info.IpAddr)
	cmd.AppendArgs("-U", info.Username)
	cmd.AppendArgs("-P", info.Password)
	cmd.AppendArgs("sol", "activate")
	tool := &IpmitoolSol{
		BaseCommand: cmd,
		Info:        info,
	}
	return tool, nil
}

func (c *IpmitoolSol) GetCommand() *exec.Cmd {
	return c.BaseCommand.GetCommand()
}

func (c IpmitoolSol) GetProtocol() string {
	return PROTOCOL_TTY
}
