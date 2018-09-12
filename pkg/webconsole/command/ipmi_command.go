package command

import (
	"fmt"
	"os/exec"

	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

type IpmiInfo struct {
	IpAddr   string `json:"ip_addr"`
	Username string `json:"username"`
	Password string `json:"password"`
	Present  bool   `json:"present"`
}

type IpmitoolSol struct {
	*BaseCommand
	Info *IpmiInfo
}

func NewIpmitoolSolCommand(info *IpmiInfo) (*IpmitoolSol, error) {
	if info.IpAddr == "" {
		return nil, fmt.Errorf("Empty host ip address")
	}
	if info.Username == "" {
		return nil, fmt.Errorf("Empty username")
	}
	if info.Password == "" {
		return nil, fmt.Errorf("Empty password")
	}
	name := o.Options.IpmitoolPath
	cmd := NewBaseCommand(name, "-I", "lanplus")
	cmd.AppendArgs("-H", info.IpAddr)
	cmd.AppendArgs("-U", info.Username)
	cmd.AppendArgs("-P", info.Password)
	cmd.AppendArgs("sol", "activate")
	tool := &IpmitoolSol{
		BaseCommand: cmd,
		Info:        info,
	}
	return tool, nil
}

func (c *IpmitoolSol) GetCommand() *exec.Cmd {
	return c.BaseCommand.GetCommand()
}

func (c IpmitoolSol) GetProtocol() string {
	return PROTOCOL_TTY
}
