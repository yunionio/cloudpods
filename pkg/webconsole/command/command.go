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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/webconsole/recorder"
)

const (
	PROTOCOL_TTY string = "tty"
	//PROTOCOL_VNC string = "vnc"
)

type ICommand interface {
	GetProtocol() string
	GetCommand() *exec.Cmd
	Cleanup() error
	Scan(d byte, send func(msg string))
	GetClientSession() *mcclient.ClientSession
	GetRecordObject() *recorder.Object
}

type BaseCommand struct {
	s    *mcclient.ClientSession
	name string
	args []string
}

func NewBaseCommand(s *mcclient.ClientSession, name string, args ...string) *BaseCommand {
	return &BaseCommand{
		s:    s,
		name: name,
		args: args,
	}
}

func (c *BaseCommand) GetClientSession() *mcclient.ClientSession {
	return c.s
}

func (c *BaseCommand) AppendArgs(args ...string) *BaseCommand {
	c.args = append(c.args, args...)
	return c
}

func (c BaseCommand) GetCommand() *exec.Cmd {
	return exec.Command(c.name, c.args...)
}

func (c BaseCommand) Scan(byte, func(msg string)) {
}

func (c BaseCommand) Cleanup() error {
	log.Infof("BaseCommand Cleanup do nothing")
	return nil
}

func (c BaseCommand) GetRecordObject() *recorder.Object {
	return nil
}
