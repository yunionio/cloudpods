package procutils

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
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
	log.Infof("Exec command: %s %v", c.Path, c.Args)
	output, err := RunCommandWithoutTimeout(c.Path, c.Args...)
	if err != nil {
		log.Errorf("Execute command %q , error: %v , output: %s", c, err, string(output))
	}
	return output, err
}

// Have default timeout 3 * time.Second
func (c *Command) RunWithTimeout() ([]byte, error) {
	output, err := RunCommandWithTimeout(c.Path, c.Args...)
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

func RunCommandWithTimeout(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	return RunCommandWithContext(ctx, name, args...)
}

func RunCommandWithContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)

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
