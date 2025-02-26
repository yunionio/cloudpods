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

package procutils

import (
	"context"
	"io"
	"strings"
	"time"

	"yunion.io/x/log"
)

var (
	Timeout = 3 * time.Second
)

type Command struct {
	path string
	args []string

	cmd       Cmd
	remoteCmd bool
}

func NewCommand(name string, args ...string) *Command {
	return &Command{
		path: name,
		args: args,
		cmd:  localExecutor.Command(name, args...),
	}
}

func NewCommandContext(ctx context.Context, name string, args ...string) *Command {
	return &Command{
		path: name,
		args: args,
		cmd:  localExecutor.CommandContext(ctx, name, args...),
	}
}

// exec remote command as far as possible
func NewRemoteCommandAsFarAsPossible(name string, args ...string) *Command {
	return &Command{
		path:      name,
		args:      args,
		cmd:       execInstance.Command(name, args...),
		remoteCmd: true,
	}
}

func NewRemoteCommandContextAsFarAsPossible(ctx context.Context, name string, args ...string) *Command {
	return &Command{
		path:      name,
		args:      args,
		cmd:       execInstance.CommandContext(ctx, name, args...),
		remoteCmd: true,
	}
}

func (c *Command) StdinPipe() (io.WriteCloser, error) {
	return c.cmd.StdinPipe()
}

func (c *Command) StdoutPipe() (io.ReadCloser, error) {
	return c.cmd.StdoutPipe()
}

func (c *Command) StderrPipe() (io.ReadCloser, error) {
	return c.cmd.StderrPipe()
}

func (c *Command) Run() error {
	log.Debugf("Exec command: %s %v", c.path, c.args)
	err := c.cmd.Run()
	if err != nil {
		log.Debugf("Execute command %q , error: %v", c, err)
	}
	return err
}

func (c *Command) Output() ([]byte, error) {
	log.Debugf("Exec command: %s %v", c.path, c.args)
	output, err := c.cmd.CombinedOutput()
	if err != nil {
		log.Debugf("Execute command %q , error: %v , output: %s", c, err, string(output))
	}
	return output, err
}

func (c *Command) Start() error {
	log.Debugf("Exec command: %s %v", c.path, c.args)
	return c.cmd.Start()
}

func (c *Command) SetEnv(envs []string) {
	c.cmd.SetEnv(envs)
}

func (c *Command) Wait() error {
	return c.cmd.Wait()
}

func (c *Command) String() string {
	ss := []string{c.path}
	ss = append(ss, c.args...)
	return strings.Join(ss, " ")
}

func (c *Command) Kill() error {
	return c.cmd.Kill()
}

func (c *Command) GetExitStatus(err error) (int, bool) {
	if c.remoteCmd {
		return _remoteExecutor.GetExitStatus(err)
	} else {
		return localExecutor.GetExitStatus(err)
	}
}
