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
	"os/exec"

	"yunion.io/x/log"
)

const (
	PROTOCOL_TTY string = "tty"
	//PROTOCOL_VNC string = "vnc"
)

type ICommand interface {
	GetProtocol() string
	GetCommand() *exec.Cmd
	Cleanup() error
	Reconnect()
	IsNeedShowInfo() bool
	ShowInfo() string
	Scan(d byte, send func(msg string))
}

type BaseCommand struct {
	name string
	args []string
}

func NewBaseCommand(name string, args ...string) *BaseCommand {
	return &BaseCommand{
		name: name,
		args: args,
	}
}

func (c *BaseCommand) AppendArgs(args ...string) *BaseCommand {
	for _, arg := range args {
		c.args = append(c.args, arg)
	}
	return c
}

func (c BaseCommand) GetCommand() *exec.Cmd {
	return exec.Command(c.name, c.args...)
}

func (c BaseCommand) Scan(byte, func(msg string)) {
	return
}

func (c BaseCommand) IsNeedShowInfo() bool {
	return false
}

func (c BaseCommand) ShowInfo() string {
	return ""
}

func (c BaseCommand) Reconnect() {
	return
}

func (c BaseCommand) Cleanup() error {
	log.Infof("BaseCommand Cleanup do nothing")
	return nil
}
