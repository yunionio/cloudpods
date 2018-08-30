package shell

import (
	"fmt"
)

type CMD struct {
	Options  interface{}
	Command  string
	Desc     string
	Callback interface{}
}

var CommandTable []CMD = make([]CMD, 0)

func R(options interface{}, command string, desc string, callback interface{}) {
	CommandTable = append(CommandTable, CMD{options, command, desc, callback})
}

func InvalidUpdateError() error {
	return fmt.Errorf("No valid update data")
}
