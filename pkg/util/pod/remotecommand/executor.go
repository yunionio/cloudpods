package remotecommand

import (
	"io"
	"net/url"
)

// RemoteExecutor defines the interface accepted by the Exec command
type RemoteExecutor interface {
	Execute(method string, url *url.URL, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue TerminalSizeQueue) error
}

// DefaultRemoteExecutor is the standard implementation of remote command execution
type DefaultRemoteExecutor struct{}

func NewDefaultRemoteExecutor() RemoteExecutor {
	return &DefaultRemoteExecutor{}
}

func (d DefaultRemoteExecutor) Execute(method string, url *url.URL, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue TerminalSizeQueue) error {
	exec, err := NewSPDYExecutor(method, url)
	if err != nil {
		return err
	}
	return exec.Stream(StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               tty,
		TerminalSizeQueue: terminalSizeQueue,
	})
}
