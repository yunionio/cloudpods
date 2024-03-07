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

package compute

import (
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
)

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(new(ContainerSpec)), func() gotypes.ISerializable {
		return new(ContainerSpec)
	})
}

const (
	CONTAINER_DEV_CPH_AMD_GPU      = "CPH_AMD_GPU"
	CONTAINER_DEV_CPH_AOSP_BINDER  = "CPH_AOSP_BINDER"
	CONTAINER_DEV_NETINT_CA_ASIC   = "NETINT_CA_ASIC"
	CONTAINER_DEV_NETINT_CA_QUADRA = "NETINT_CA_QUADRA"
	CONTAINER_DEV_NVIDIA_GPU       = "NVIDIA_GPU"
)

const (
	CONTAINER_STORAGE_LOCAL_RAW = "local_raw"
)

const (
	CONTAINER_STATUS_PULLING_IMAGE      = "pulling_image"
	CONTAINER_STATUS_PULL_IMAGE_FAILED  = "pull_image_failed"
	CONTAINER_STATUS_PULLED_IMAGE       = "pulled_image"
	CONTAINER_STATUS_CREATING           = "creating"
	CONTAINER_STATUS_CREATE_FAILED      = "create_failed"
	CONTAINER_STATUS_STARTING           = "starting"
	CONTAINER_STATUS_START_FAILED       = "start_failed"
	CONTAINER_STATUS_STOPPING           = "stopping"
	CONTAINER_STATUS_STOP_FAILED        = "stop_failed"
	CONTAINER_STATUS_SYNC_STATUS        = "sync_status"
	CONTAINER_STATUS_SYNC_STATUS_FAILED = "sync_status_failed"
	CONTAINER_STATUS_UNKNOWN            = "unknown"
	CONTAINER_STATUS_CREATED            = "created"
	CONTAINER_STATUS_EXITED             = "exited"
	CONTAINER_STATUS_RUNNING            = "running"
	CONTAINER_STATUS_DELETING           = "deleting"
	CONTAINER_STATUS_DELETE_FAILED      = "delete_failed"
)

const (
	CONTAINER_METADATA_CRI_ID = "cri_id"
)

type ContainerSpec struct {
	apis.ContainerSpec
	// Mounts for the container.
	// Mounts []*ContainerMount `json:"mounts"`
	Devices []*ContainerDevice `json:"devices"`
}

func (c *ContainerSpec) String() string {
	return jsonutils.Marshal(c).String()
}

func (c *ContainerSpec) IsZero() bool {
	if reflect.DeepEqual(*c, ContainerSpec{}) {
		return true
	}
	return false
}

type ContainerCreateInput struct {
	apis.VirtualResourceCreateInput

	GuestId string        `json:"guest_id"`
	Spec    ContainerSpec `json:"spec"`
	// swagger:ignore
	SkipTask bool `json:"skip_task"`
}

type ContainerListInput struct {
	apis.VirtualResourceListInput
}

type ContainerStopInput struct {
	Timeout int `json:"timeout"`
}

type ContainerSyncStatusResponse struct {
	Status string `json:"status"`
}

type ContainerDesc struct {
	Id   string         `json:"id"`
	Name string         `json:"name"`
	Spec *ContainerSpec `json:"spec"`
}

type ContainerHostDevice struct {
	// Path of the device within the container.
	ContainerPath string `json:"container_path"`
	// Path of the device on the host.
	HostPath string `json:"host_path"`
	// Cgroups permissions of the device, candidates are one or more of
	// * r - allows container to read from the specified device.
	// * w - allows container to write to the specified device.
	// * m - allows container to create device files that do not yet exist.
	Permissions string `json:"permissions"`
}

type ContainerIsolatedDevice struct {
	Index *int   `json:"index"`
	Id    string `json:"id"`
}

type ContainerDevice struct {
	Type           apis.ContainerDeviceType `json:"type"`
	IsolatedDevice *ContainerIsolatedDevice `json:"isolated_device"`
	Host           *ContainerHostDevice     `json:"host"`
}
