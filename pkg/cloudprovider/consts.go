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

package cloudprovider

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	CloudVMStatusRunning      = "running"
	CloudVMStatusStopping     = "stopping"
	CloudVMStatusSuspend      = "suspend"
	CloudVMStatusStopped      = "stopped"
	CloudVMStatusChangeFlavor = "change_flavor"
	CloudVMStatusDeploying    = "deploying"
	CloudVMStatusOther        = "other"

	ErrNotFound       = errors.ErrNotFound
	ErrDuplicateId    = errors.ErrDuplicateId
	ErrInvalidStatus  = errors.ErrInvalidStatus
	ErrTimeout        = errors.ErrTimeout
	ErrNotImplemented = errors.ErrNotImplemented
	ErrNotSupported   = errors.ErrNotSupported

	ErrInvalidProvider     = httperrors.ErrInvalidProvider
	ErrNoBalancePermission = httperrors.ErrNoBalancePermission
)

const (
	CLOUD_CAPABILITY_PROJECT      = "project"
	CLOUD_CAPABILITY_COMPUTE      = "compute"
	CLOUD_CAPABILITY_LOADBALANCER = "loadbalancer"
	CLOUD_CAPABILITY_OBJECTSTORE  = "objectstore"
	CLOUD_CAPABILITY_RDS          = "rds"
	CLOUD_CAPABILITY_CACHE        = "cache"
	CLOUD_CAPABILITY_EVENT        = "event"
)
