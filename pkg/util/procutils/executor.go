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
	"os/exec"
	"syscall"

	"yunion.io/x/executor/client"
)

var (
	execInstance    Executor
	localExecutor   = new(defaultExecutor)
	_remoteExecutor = new(remoteExecutor)
)

func init() {
	execInstance = localExecutor
}

func SetRemoteExecutor() {
	execInstance = _remoteExecutor
}

type Cmd interface {
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	CombinedOutput() ([]byte, error)
	Start() error
	Wait() error
	Run() error
	Kill() error
	SetEnv([]string)
}

type Executor interface {
	CommandContext(ctx context.Context, name string, args ...string) Cmd
	Command(name string, args ...string) Cmd

	GetExitStatus(err error) (int, bool)
}

type defaultCmd struct {
	*exec.Cmd
}

func (c *defaultCmd) Kill() error {
	return c.Process.Kill()
}

func (c *defaultCmd) SetEnv(envs []string) {
	c.Env = append(c.Env, envs...)
}

type defaultExecutor struct{}

func (e *defaultExecutor) Command(name string, args ...string) Cmd {
	cmd := exec.Command(name, args...)
	cmdSetSid(cmd)
	cmdSetEnv(cmd)
	return &defaultCmd{cmd}
}

func (e *defaultExecutor) CommandContext(ctx context.Context, name string, args ...string) Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmdSetSid(cmd)
	cmdSetEnv(cmd)
	return &defaultCmd{cmd}
}

func (e *defaultExecutor) GetExitStatus(err error) (int, bool) {
	if exiterr, ok := err.(*exec.ExitError); ok {
		ws := exiterr.Sys().(syscall.WaitStatus)
		return ws.ExitStatus(), true
	} else {
		return 0, false
	}
}

type remoteCmd struct {
	*client.Cmd
}

func (c *remoteCmd) SetEnv(envs []string) {
	c.Env = append(c.Env, envs...)
}

type remoteExecutor struct{}

func (e *remoteExecutor) Command(name string, args ...string) Cmd {
	cmd := client.Command(name, args...)
	remoteCmdSetEnv(cmd)
	return &remoteCmd{cmd}
}

func (e *remoteExecutor) CommandContext(ctx context.Context, name string, args ...string) Cmd {
	cmd := client.CommandContext(ctx, name, args...)
	remoteCmdSetEnv(cmd)
	return &remoteCmd{cmd}
}

func (e *remoteExecutor) GetExitStatus(err error) (int, bool) {
	if exiterr, ok := err.(*client.ExitError); ok {
		ws := exiterr.Sys().(syscall.WaitStatus)
		return ws.ExitStatus(), true
	} else {
		return 0, false
	}
}
