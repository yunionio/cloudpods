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
	"errors"
)

const (
	CloudVMStatusRunning      = "running"
	CloudVMStatusStopping     = "stopping"
	CloudVMStatusSuspend      = "suspend"
	CloudVMStatusStopped      = "stopped"
	CloudVMStatusChangeFlavor = "change_flavor"
	CloudVMStatusDeploying    = "deploying"
	CloudVMStatusOther        = "other"
)

var ErrNotFound = errors.New("id not found")
var ErrDuplicateId = errors.New("duplicate id")
var ErrInvalidStatus = errors.New("invalid status")
var ErrTimeout = errors.New("timeout")
var ErrNotImplemented = errors.New("Not implemented")
var ErrNotSupported = errors.New("Not supported")
var ErrInvalidProvider = errors.New("Invalid provider")
var ErrNoBalancePermission = errors.New("No balance permission")
