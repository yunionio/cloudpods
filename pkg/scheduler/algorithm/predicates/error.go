// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	ErrRequireMvs                             = `require mvs`
	ErrRequireNoMvs                           = `require not mvs`
	ErrHostIsSpecifiedForMigration            = `host_id specified for migration`
	ErrHostCpuModelIsNotMatchForLiveMigrate   = `host cpu mode not match for live migrate`
	ErrHostCpuMicrocodeNotMatchForLiveMigrate = `host cpu microcode not match for live migrate`
	ErrMoreThanOneSizeUnspecificSplit         = `more than 1 size unspecific split`
	ErrNoMoreSpaceForUnspecificSplit          = `no more space for an unspecific split`
	ErrSubtotalOfSplitExceedsDiskSize         = `subtotal of split exceeds disk size`
	ErrBaremetalHasAlreadyBeenOccupied        = `baremetal has already been occupied`
	ErrPrepaidHostOccupied                    = `prepaid host occupied`

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

type FailReason struct {
	Reason string
	Type   string
}

func (r FailReason) GetReason() string {
	return r.Reason
}

func (r FailReason) GetType() string {
	return r.Type
}

const (
	NetworkPrivate   = "network_private"
	NetworkPublic    = "network_public"
	NetworkTypeMatch = "network_type"
	NetworkMatch     = "network_match"
	NetworkWire      = "network_wire"
	NetworkOwner     = "network_owner"
	NetworkDomain    = "network_domain"
	NetworkRange     = "network_range"
	NetworkFreeCount = "network_free_count"
	NetworkPort      = "network_port"

	StorageEnable = "storage_status"
	StorageMatch  = "storage_match"
	StorageType   = "storage_type"
)
