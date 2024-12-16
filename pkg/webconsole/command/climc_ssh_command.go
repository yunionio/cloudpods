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
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/webconsole/helper"
)

type ClimcSshCommand struct {
	*BaseCommand
	Info    *webconsole.ClimcSshInfo
	s       *mcclient.ClientSession
	keyFile string
	buffer  []byte
}

func NewClimcSshCommand(info *webconsole.ClimcSshInfo, s *mcclient.ClientSession) (*ClimcSshCommand, error) {
	if info.IpAddr == "" {
		return nil, fmt.Errorf("Empty host ip address")
	}
	if info.Username == "" {
		return nil, fmt.Errorf("Empty username")
	}
	privateKey, err := helper.GetValidPrivateKey(info.IpAddr, 22, info.Username, "")
	if err != nil {
		return nil, errors.Wrap(err, "get cloud admin private key")
	}
	file, err := ioutil.TempFile("", fmt.Sprintf("id_rsa.%s.", info.IpAddr))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	filename := file.Name()
	{
		err = os.Chmod(filename, 0600)
		if err != nil {
			return nil, err
		}
		_, err = file.Write([]byte(privateKey))
		if err != nil {
			return nil, err
		}
	}
	env := map[string]string{
		"OS_AUTH_TOKEN":           s.GetToken().GetTokenString(),
		"OS_PROJECT_NAME":         s.GetProjectName(),
		"OS_PROJECT_DOMAIN":       s.GetProjectDomain(),
		"YUNION_USE_CACHED_TOKEN": "false",
		"OS_TRY_TERM_WIDTH":       "false",
	}
	if len(info.Env) != 0 {
		env = info.Env
	}
	envCmd := ""
	for k, v := range env {
		envCmd = fmt.Sprintf("%s export %s=%s", envCmd, k, v)
	}
	execCmd := "exec bash"
	if info.Command != "" {
		execCmd = info.Command
		execCmd = fmt.Sprintf("%s %s", execCmd, strings.Join(info.Args, " "))
	}
	sshArgs := []string{
		"-t", // force pseudo-terminal allocation
		"-o", "StrictHostKeyChecking=no",
		"-i", filename,
		fmt.Sprintf("%s@%s", info.Username, info.IpAddr),
		fmt.Sprintf("'%s && %s'", envCmd, execCmd),
	}
	sshCmd := fmt.Sprintf("ssh %s", strings.Join(sshArgs, " "))
	args := []string{"-c", sshCmd}
	bCmd := NewBaseCommand(s, "bash", args...)
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
