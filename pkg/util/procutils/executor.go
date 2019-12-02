package procutils

import (
	"context"
	"io"
	"os/exec"
	"syscall"

	"yunion.io/x/onecloud/pkg/executor/client"
)

var execInstance Executor

func init() {
	execInstance = new(defaultExecutor)
}

func SetSocketExecutor() {
	execInstance = new(socketExecutor)
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

type defaultExecutor struct{}

func (e *defaultExecutor) Command(name string, args ...string) Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	return &defaultCmd{cmd}
}

func (e *defaultExecutor) CommandContext(ctx context.Context, name string, args ...string) Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
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

type socketExecutor struct{}

func (e *socketExecutor) Command(name string, args ...string) Cmd {
	cmd := client.Command(name, args...)
	return cmd
}

func (e *socketExecutor) CommandContext(ctx context.Context, name string, args ...string) Cmd {
	cmd := client.CommandContext(ctx, name, args...)
	return cmd
}

func (e *socketExecutor) GetExitStatus(err error) (int, bool) {
	if exiterr, ok := err.(*client.ExitError); ok {
		ws := exiterr.Sys().(syscall.WaitStatus)
		return ws.ExitStatus(), true
	} else {
		return 0, false
	}
}
