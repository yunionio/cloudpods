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
