package command

import (
	"fmt"
	"net"
	"os/exec"
	"time"

	"yunion.io/x/log"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

type SSHtoolSol struct {
	*BaseCommand
	IP       string
	Username string
	reTry    int
	showInfo string
}

func NewSSHtoolSolCommand(ip string) (*SSHtoolSol, error) {
	if conn, err := net.DialTimeout("tcp", ip+":22", time.Second*2); err != nil {
		return nil, fmt.Errorf("IPAddress %s not accessable", ip)
	} else {
		conn.Close()
		return &SSHtoolSol{
			BaseCommand: nil,
			IP:          ip,
			Username:    "",
			reTry:       0,
			showInfo:    fmt.Sprintf("%s login:", ip),
		}, nil
	}
}

func (c *SSHtoolSol) GetCommand() *exec.Cmd {
	return nil
}

func (c *SSHtoolSol) Cleanup() error {
	log.Infof("SSHtoolSol Cleanup do nothing")
	return nil
}

func (c *SSHtoolSol) GetProtocol() string {
	return PROTOCOL_TTY
}

func (c *SSHtoolSol) GetData(data string) (isShow bool, ouput string, command string) {
	if len(c.Username) == 0 {
		if len(data) == 0 {
			//用户名不能为空
			return true, c.showInfo, ""
		}
		c.Username = data
		return false, "Password:", ""
	} else {
		return true, "", fmt.Sprintf("%s -p %s %s %s@%s", o.Options.SshpassToolPath, data, o.Options.SshToolPath, c.Username, c.IP)
	}
}

func (c *SSHtoolSol) ShowInfo() string {
	c.Username = ""
	c.reTry++
	if c.reTry == 3 {
		c.reTry = 0
		//清屏
		time.Sleep(1 * time.Second)
		return "\033c " + c.showInfo
	}
	return c.showInfo
}
