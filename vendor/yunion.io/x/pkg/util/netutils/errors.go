package netutils

import (
	"github.com/pkg/errors"
)

var (
	ErrInvalidNumber  = errors.New("invalid number")
	ErrOutOfRange     = errors.New("ip number out of range [0-255]")
	ErrInvalidIPAddr  = errors.New("invalid ip address")
	ErrInvalidMask    = errors.New("invalid mask")
	ErrOutOfRangeMask = errors.New("out of range masklen [0-32]")
)
