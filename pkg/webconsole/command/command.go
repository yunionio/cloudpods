package command

import (
	"os/exec"

	"yunion.io/x/log"
)

const (
	PROTOCOL_TTY string = "tty"
	//PROTOCOL_VNC string = "vnc"
)

type ICommand interface {
	GetProtocol() string
	GetCommand() *exec.Cmd
	Cleanup() error
	GetData(string) (isShow bool, ouput string, command string)
	Connect() error
	ShowInfo() string
}

type BaseCommand struct {
	name string
	args []string
}

func NewBaseCommand(name string, args ...string) *BaseCommand {
	return &BaseCommand{
		name: name,
		args: args,
	}
}

func (c *BaseCommand) AppendArgs(args ...string) *BaseCommand {
	for _, arg := range args {
		c.args = append(c.args, arg)
	}
	return c
}

func (c BaseCommand) GetCommand() *exec.Cmd {
	return exec.Command(c.name, c.args...)
}

func (c BaseCommand) Connect() error {
	return nil
}

func (c BaseCommand) GetData(comand string) (isShow bool, ouput string, command string) {
	return true, "", ""
}

func (c BaseCommand) ShowInfo() string {
	return ""
}

func (c BaseCommand) Cleanup() error {
	log.Infof("BaseCommand Cleanup do nothing")
	return nil
}
