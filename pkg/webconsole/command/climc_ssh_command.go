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
	"os"
	"os/exec"
	"regexp"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/webconsole/helper"
)

var (
	usernameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	envKeyRe   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// shellQuote quotes s as a single POSIX shell word, so that it stays literal
// data when interpreted by the remote shell.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// buildRemoteCmd builds the command executed by the remote shell. Every
// interpolated value is passed through shellQuote so it cannot escape into
// shell syntax (command injection).
func buildRemoteCmd(env map[string]string, cmd string, args []string) (string, error) {
	parts := make([]string, 0, len(env)+1)
	for k, v := range env {
		if !envKeyRe.MatchString(k) {
			return "", fmt.Errorf("invalid env key %q", k)
		}
		parts = append(parts, fmt.Sprintf("export %s=%s", k, shellQuote(v)))
	}
	if cmd != "" {
		tokens := append([]string{cmd}, args...)
		quoted := make([]string, len(tokens))
		for i, token := range tokens {
			quoted[i] = shellQuote(token)
		}
		parts = append(parts, strings.Join(quoted, " "))
	} else {
		parts = append(parts, "exec bash")
	}
	return strings.Join(parts, " && "), nil
}

type ClimcSshCommand struct {
	*BaseCommand
	Info    *webconsole.ClimcSshInfo
	s       *mcclient.ClientSession
	keyFile string
	buffer  []byte
}

func NewClimcSshCommand(info *webconsole.ClimcSshInfo, s *mcclient.ClientSession) (*ClimcSshCommand, error) {
	if info.Username == "" {
		return nil, fmt.Errorf("Empty username")
	}
	if !usernameRe.MatchString(info.Username) {
		return nil, fmt.Errorf("Invalid username %q", info.Username)
	}
	targetIp := helper.FetchClimcTargetIp()
	privateKey, err := helper.GetValidPrivateKey(targetIp, 22, info.Username, "")
	if err != nil {
		return nil, errors.Wrap(err, "get cloud admin private key")
	}
	file, err := os.CreateTemp("", fmt.Sprintf("id_rsa.%s.", targetIp))
	if err != nil {
		return nil, err
	}
	filename := file.Name()
	err = func() error {
		defer file.Close()
		err = os.Chmod(filename, 0600)
		if err != nil {
			return err
		}
		_, err = file.Write([]byte(privateKey))
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return nil, err
	}
	env := map[string]string{
		"OS_AUTH_TOKEN":           s.GetToken().GetTokenString(),
		"OS_PROJECT_NAME":         s.GetProjectName(),
		"OS_PROJECT_DOMAIN":       s.GetProjectDomain(),
		"YUNION_USE_CACHED_TOKEN": "false",
		"OS_TRY_TERM_WIDTH":       "false",
		"GOMAXPROCS":              "2",
		"OS_USERNAME":             "",
		"OS_PASSWORD":             "",
		"OS_DOMAIN_NAME":          "",
		"OS_ACCESS_KEY":           "",
		"OS_SECRET_KEY":           "",
	}
	if len(info.Env) != 0 {
		env = info.Env
	}
	remoteCmd, err := buildRemoteCmd(env, info.Command, info.Args)
	if err != nil {
		return nil, err
	}
	// argv is passed directly to ssh without a shell, so user input cannot
	// escape into local command execution
	sshArgs := []string{
		"-t", // force pseudo-terminal allocation
		"-o", "StrictHostKeyChecking=no",
		"-i", filename,
		fmt.Sprintf("%s@%s", info.Username, targetIp),
		remoteCmd,
	}
	bCmd := NewBaseCommand(s, "ssh", sshArgs...)
	cmd := &ClimcSshCommand{
		BaseCommand: bCmd,
		Info:        info,
		s:           s,
		keyFile:     filename,
		buffer:      []byte{},
	}
	return cmd, nil
}

func (c ClimcSshCommand) GetCommand() *exec.Cmd {
	cmd := c.BaseCommand.GetCommand()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	return cmd
}

func (c ClimcSshCommand) GetInstanceName() string {
	if c.Info.DisplayInfo == nil {
		return ""
	}
	return c.Info.DisplayInfo.InstanceName
}

func (c ClimcSshCommand) GetIPs() []string {
	if c.Info.DisplayInfo == nil {
		return nil
	}
	return c.Info.DisplayInfo.IPs
}

func (c ClimcSshCommand) GetProtocol() string {
	return PROTOCOL_TTY
}

func (c ClimcSshCommand) Cleanup() error {
	if len(c.keyFile) > 0 {
		os.Remove(c.keyFile)
		c.keyFile = ""
	}
	return nil
}

func (c *ClimcSshCommand) Scan(d byte, send func(msg string)) {
	switch d {
	case '\r': // 换行
		send("\r\n")
		c.buffer = []byte{}
	case '\u007f': // 退格
		if len(c.buffer) > 0 {
			c.buffer = c.buffer[:len(c.buffer)-1]
			send("\b \b")
		}
	default:
		c.buffer = append(c.buffer, d)
	}
}
