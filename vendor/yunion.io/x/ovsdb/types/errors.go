package types

import (
	"errors"
)

var (
	ErrUnknownColumn = errors.New("unknown column")
	ErrUnknownTable  = errors.New("unknown table")

	ErrBadType = errors.New("bad type")
)
