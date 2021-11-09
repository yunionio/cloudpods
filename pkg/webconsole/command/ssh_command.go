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
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/ssh"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

type SSHtoolSol struct {
	*BaseCommand
	IP           string
	Port         int
	username     string
	password     string
	failed       int
	showInfo     string
	keyFile      string
	buffer       []byte
	needShowInfo bool
}

func getCommand(ctx context.Context, userCred mcclient.TokenCredential, ip string, port int) (string, *BaseCommand, error) {
	if !o.Options.EnableAutoLogin {
		return "", nil, nil
	}
	s := auth.GetAdminSession(ctx, o.Options.Region, "v2")
	key, err := modules.Sshkeypairs.GetById(s, userCred.GetProjectId(), jsonutils.Marshal(map[string]bool{"admin": true}))
	if err != nil {
		return "", nil, err
	}
	privKey, err := key.GetString("private_key")
	if err != nil {
		return "", nil, err
	}
	file, err := ioutil.TempFile("", fmt.Sprintf("id_rsa.%s.", ip))
	if err != nil {
		return "", nil, err
	}
	defer file.Close()
	filename := file.Name()
	{
		err = os.Chmod(filename, 0600)
		if err != nil {
			return "", nil, err
		}
		_, err = file.Write([]byte(privKey))
		if err != nil {
			return "", nil, err
		}
	}

	// try ssh without password login
	var cmd *BaseCommand
	user := ansible.PUBLIC_CLOUD_ANSIBLE_USER
	if _, err := ssh.NewClient(ip, port, user, "", privKey); err != nil {
		log.Warningf("try use %s without password login error: %v", user, err)
		return "", nil, nil
	} else {
		cmd = NewBaseCommand(o.Options.SshToolPath)
		cmd.AppendArgs("-i", filename)
		cmd.AppendArgs("-q")
		cmd.AppendArgs("-o", "StrictHostKeyChecking=no")
		cmd.AppendArgs("-o", "GlobalKnownHostsFile=/dev/null")
		cmd.AppendArgs("-o", "UserKnownHostsFile=/dev/null")
		cmd.AppendArgs("-o", "PasswordAuthentication=no")
		cmd.AppendArgs("-o", "BatchMode=yes") // 强制禁止密码登录
		cmd.AppendArgs("-p", fmt.Sprintf("%d", port))
		cmd.AppendArgs(fmt.Sprintf("%s@%s", user, ip))
	}

	return filename, cmd, nil
}

func NewSSHtoolSolCommand(ctx context.Context, userCred mcclient.TokenCredential, ip string, query jsonutils.JSONObject) (*SSHtoolSol, error) {
	port := 22
	if query != nil {
		if _port, _ := query.Int("webconsole", "port"); _port != 0 {
			port = int(_port)
		}
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Second*2)
	if err != nil {
		return nil, fmt.Errorf("IPAddress %s:%d not accessible", ip, port)
	}
	defer conn.Close()

	keyFile, cmd, err := getCommand(ctx, userCred, ip, port)
	if err != nil {
		log.Errorf("getCommand error: %v", err)
	}

	return &SSHtoolSol{
		BaseCommand:  cmd,
		IP:           ip,
		Port:         port,
		username:     "",
		failed:       0,
		showInfo:     fmt.Sprintf("%s login: ", ip),
		keyFile:      keyFile,
		buffer:       []byte{},
		needShowInfo: true,
	}, nil
}

func (c *SSHtoolSol) GetCommand() *exec.Cmd {
	if c.BaseCommand != nil {
		cmd := c.BaseCommand.GetCommand()
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
		return cmd
	}
	if len(c.username) > 0 && len(c.password) > 0 {
		args := []string{
			o.Options.SshpassToolPath, "-p", c.password,
			o.Options.SshToolPath, "-p", fmt.Sprintf("%d", c.Port), fmt.Sprintf("%s@%s", c.username, c.IP),
			"-oGlobalKnownHostsFile=/dev/null",
			"-oUserKnownHostsFile=/dev/null",
			"-oStrictHostKeyChecking=no",
			"-oPreferredAuthentications=password,keyboard-interactive",
			"-oPubkeyAuthentication=no", // 密码登录时,避免搜寻秘钥登录
			"-oNumberOfPasswordPrompts=1",
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
		return cmd
	}
	return nil
}

func (c *SSHtoolSol) Cleanup() error {
	if len(c.keyFile) > 0 {
		os.Remove(c.keyFile)
		c.keyFile = ""
	}
	return nil
}

func (c *SSHtoolSol) GetProtocol() string {
	return PROTOCOL_TTY
}

func (c *SSHtoolSol) Connect() error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.IP, c.Port), time.Second*2)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func (c *SSHtoolSol) Scan(d byte, send func(msg string)) {
	switch d {
	case '\r': // 换行
		send("\r\n")
		if len(c.username) == 0 {
			c.username = string(c.buffer)
			c.needShowInfo = true
		} else if len(c.password) == 0 {
			c.password = string(c.buffer)
		}
		c.buffer = []byte{}
	case '\u007f': // 退格
		if len(c.buffer) > 1 {
			c.buffer = c.buffer[:len(c.buffer)-1]
		}
		send("\b \b")
	default:
		c.buffer = append(c.buffer, d)
		if len(c.username) == 0 {
			send(string(d))
		}
	}
	return
}

func (c *SSHtoolSol) Reconnect() {
	c.needShowInfo, c.username, c.password = true, "", ""
}

func (c *SSHtoolSol) IsNeedShowInfo() bool {
	return c.needShowInfo
}

func (c *SSHtoolSol) ShowInfo() string {
	c.BaseCommand = nil
	c.needShowInfo = false
	if len(c.username) == 0 {
		if c.failed >= 3 {
			c.failed = 0
			return "\033c " + c.showInfo // 清屏
		}
		c.failed++
		return c.showInfo
	}
	if len(c.password) == 0 {
		return "Password:"
	}
	return ""
}
