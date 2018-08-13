package cloudprovider

import (
	"errors"
)

const (
	CloudVMStatusRunning = "running"
	CloudVMStatusSuspend = "suspend"
	CloudVMStatusStopped = "stopped"
	CloudVMStatusOther   = "other"
)

var ErrNotFound = errors.New("id not found")
var ErrDuplicateId = errors.New("duplicate id")
var ErrInvalidStatus = errors.New("invalid status")
var ErrTimeout = errors.New("timeout")
var ErrNotImplemented = errors.New("Not implemented")
var ErrNotSupported = errors.New("Not supported")
