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
	"bytes"
	"context"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"yunion.io/x/log"
)

var (
	Timeout = 3 * time.Second
)

type Command struct {
	Path string
	Args []string
}

func NewCommand(name string, args ...string) *Command {
	return &Command{
		Path: name,
		Args: args,
	}
}

func ParseOutput(output []byte) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func Run(name string, args ...string) ([]string, error) {
	ret, err := NewCommand(name, args...).Run()
	if err != nil {
		return nil, err
	}
	return ParseOutput(ret), nil
}

// Doesn't have timeout
func (c *Command) Run() ([]byte, error) {
	log.Debugf("Exec command: %s %v", c.Path, c.Args)
	output, err := RunCommandWithoutTimeout(c.Path, c.Args...)
	if err != nil {
		log.Errorf("Execute command %q , error: %v , output: %s", c, err, string(output))
	}
	return output, err
}

// Have default timeout 3 * time.Second
func (c *Command) RunWithTimeout(timeout time.Duration) ([]byte, error) {
	if timeout <= 0 {
		timeout = Timeout
	}
	output, err := RunCommandWithTimeout(timeout, c.Path, c.Args...)
	if err != nil {
		log.Errorf("Execute command %q , error: %v , output: %s", c, err, string(output))
	}
	return output, err
}

func (c *Command) RunWithContext(ctx context.Context) ([]byte, error) {
	output, err := RunCommandWithContext(ctx, c.Path, c.Args...)
	if err != nil {
		log.Errorf("Execute command %q , error: %v , output: %s", c, err, string(output))
	}
	return output, err
}

func (c *Command) String() string {
	ss := []string{c.Path}
	ss = append(ss, c.Args...)
	return strings.Join(ss, " ")
}

func RunCommandWithoutTimeout(name string, args ...string) ([]byte, error) {
	return RunCommandWithContext(context.Background(), name, args...)
}

func RunCommandWithTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return RunCommandWithContext(ctx, name, args...)
}

func RunCommandWithContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		return buf.Bytes(), err
	}

	if err := cmd.Wait(); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}

// https://gist.github.com/kylelemons/1525278
func Pipeline(cmds ...*exec.Cmd) ([]byte, []byte, error) {
	// Requires at least one command
	if len(cmds) < 1 {
		return nil, nil, nil
	}

	// Collect the output from the command(s)
	var output bytes.Buffer
	var stderr bytes.Buffer

	last := len(cmds) - 1
	for i, cmd := range cmds[:last] {
		var err error
		// Connect each command's stdin to the previous command's stdout
		if cmds[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
			return nil, nil, err
		}
		// Connect each command's stderr to a buffer
		cmd.Stderr = &stderr
	}

	// Connect the output and error for the last command
	cmds[last].Stdout, cmds[last].Stderr = &output, &stderr

	// Start each command
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}

	}

	// Wait for each command to complete
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}

	}

	// Return the pipeline output and the collected standard error
	return output.Bytes(), stderr.Bytes(), nil
}
