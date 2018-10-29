package shell

import (
	"errors"
)

type CMD struct {
	Options  interface{}
	Command  string
	Desc     string
	Callback interface{}
}

var CommandTable []CMD = make([]CMD, 0)

var ErrEmtptyUpdate = errors.New("No valid update data")

func R(options interface{}, command string, desc string, callback interface{}) {
	CommandTable = append(CommandTable, CMD{options, command, desc, callback})
}

func InvalidUpdateError() error {
	return ErrEmtptyUpdate
}
