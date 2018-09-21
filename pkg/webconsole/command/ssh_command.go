package command

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/pkg/utils"
)

type Metadata struct {
	LoginAccount string `json:"login_account"`
	LoginKey     string `json:"login_key"`
}

type SSHInfo struct {
	ID       string   `json:"id"`
	Metadata Metadata `json:"metadata"`
	Eip      string   `json:"eip"`
	IPs      string   `json:"ips"`
	Keypaire string   `json:"keypair"`
	OsType   string   `json:"os_type"`
}

type SSHtoolSol struct {
	*BaseCommand
	Info *SSHInfo
}

func NewSSHtoolSolCommand(info *SSHInfo) (*SSHtoolSol, error) {
	if info.IPs == "" {
		return nil, fmt.Errorf("Empty server ip address")
	}
	if info.Metadata.LoginAccount == "" {
		return nil, fmt.Errorf("Empty username")
	}
	if len(info.Keypaire) != 0 {
		return nil, fmt.Errorf("Not support private_key login")
	}
	if info.OsType != "Linux" {
		return nil, fmt.Errorf("Not support login for %s", info.OsType)
	}
	args := ""
	if info.Eip != "" {
		args = fmt.Sprintf("%s@%s", info.Metadata.LoginAccount, info.Eip)
	} else {
		for _, ip := range strings.Split(info.IPs, ",") {
			conn, err := net.Dial("tcp", fmt.Sprintf("%s:22", ip))
			if err == nil {
				args = fmt.Sprintf("%s@%s", info.Metadata.LoginAccount, ip)
				break
			}
			defer conn.Close()
		}
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("failed find usable connection ip address")
	}
	cmd := NewBaseCommand(o.Options.SSHtoolPath)
	if info.Metadata.LoginKey != "" {
		cmd := NewBaseCommand(o.Options.SSHtoolPath)
		if passwd, err := utils.DescryptAESBase64(info.ID, info.Metadata.LoginKey); err != nil {
			return nil, err
		} else {
			cmd.AppendArgs("-p", passwd)
		}
	}
	cmd.AppendArgs(args)
	tool := &SSHtoolSol{
		BaseCommand: cmd,
		Info:        info,
	}
	return tool, nil
}

func (c *SSHtoolSol) GetCommand() *exec.Cmd {
	return c.BaseCommand.GetCommand()
}

func (c SSHtoolSol) GetProtocol() string {
	return PROTOCOL_TTY
}
