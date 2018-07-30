package predicates

import (
	"fmt"
)

// Here are all the errors that may appear in the preselection predicates.
const (
	ErrServerTypeIsNotMatch = `server type is not match`
	ErrExitIsNotMatch       = `exit is not match`
	ErrWireIsNotMatch       = `wire is not match`

	ErrNoPorts               = `no ports`
	ErrNotOwner              = `not owner`
	ErrNoEnoughStorage       = `no enough storage`
	ErrNoAvailableNetwork    = `no available network on this host`
	ErrNoEnoughAvailableGPUs = `no enough available GPUs`
	ErrNotSupportNest        = `nested function not supported`

	ErrRequireMvs                      = `require mvs`
	ErrRequireNoMvs                    = `require not mvs`
	ErrHostIsSpecifiedForMigration     = `host_id specified for migration`
	ErrMoreThanOneSizeUnspecificSplit  = `more than 1 size unspecific split`
	ErrNoMoreSpaceForUnspecificSplit   = `no more space for an unspecific split`
	ErrSubtotalOfSplitExceedsDiskSize  = `subtotal of split exceeds disk size`
	ErrBaremetalHasAlreadyBeenOccupied = `baremetal has already been occupied`

	ErrUnknown = `unknown error`
)

// InsufficientResourceError is an error type that indicates what kind of resource limit is
// hit and caused the unfitting failure.
type InsufficientResourceError struct {
	// resourceName is the name of the resource that is insufficient
	ResourceName string
	requested    int64
	total        int64
	free         int64
}

func NewInsufficientResourceError(resourceName string, requested, total, free int64) *InsufficientResourceError {
	return &InsufficientResourceError{
		ResourceName: resourceName,
		requested:    requested,
		total:        total,
		free:         free,
	}
}

func (ire *InsufficientResourceError) Error() string {
	return fmt.Sprintf("no enough resource: %s, requested: %d, total: %d, free: %d",
		ire.ResourceName, ire.requested, ire.total, ire.free)
}

func (ire *InsufficientResourceError) GetReason() string {
	return ire.Error()
}

type UnexceptedResourceError struct {
	message string
}

func NewUnexceptedResourceError(message string) *UnexceptedResourceError {
	return &UnexceptedResourceError{
		message: message,
	}
}

func Error(message string) *UnexceptedResourceError {
	return NewUnexceptedResourceError(message)
}

func (ure *UnexceptedResourceError) Error() string {
	return ure.message
}

func (ure *UnexceptedResourceError) GetReason() string {
	return ure.Error()
}
