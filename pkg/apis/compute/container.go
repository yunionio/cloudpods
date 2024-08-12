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
	CONTAINER_DEV_NVIDIA_MPS       = "NVIDIA_MPS"
	CONTAINER_DEV_ASCEND_NPU       = "ASCEND_NPU"
	CONTAINER_DEV_VASTAITECH_GPU   = "VASTAITECH_GPU"
)

var (
	CONTAINER_GPU_TYPES = []string{
		CONTAINER_DEV_CPH_AMD_GPU,
		CONTAINER_DEV_NVIDIA_GPU,
		CONTAINER_DEV_NVIDIA_MPS,
		CONTAINER_DEV_VASTAITECH_GPU,
	}
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
	CONTAINER_STATUS_SAVING_IMAGE       = "saving_image"
	CONTAINER_STATUS_SAVE_IMAGE_FAILED  = "save_image_failed"
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
	// for health check
	CONTAINER_STATUS_PROBING      = "probing"
	CONTAINER_STATUS_PROBE_FAILED = "probe_failed"
)

const (
	CONTAINER_METADATA_CRI_ID           = "cri_id"
	CONTAINER_METADATA_RELEASED_DEVICES = "released_devices"
)

type ContainerSpec struct {
	apis.ContainerSpec
	// Volume mounts
	VolumeMounts []*apis.ContainerVolumeMount `json:"volume_mounts"`
	Devices      []*ContainerDevice           `json:"devices"`
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

type ContainerUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput
	Spec ContainerSpec `json:"spec"`
}

type ContainerListInput struct {
	apis.VirtualResourceListInput
	GuestId string `json:"guest_id"`
}

type ContainerStopInput struct {
	Timeout int `json:"timeout"`
}

type ContainerSyncStatusResponse struct {
	Status string `json:"status"`
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

type ContainerSaveVolumeMountToImageInput struct {
	Name         string `json:"name"`
	GenerateName string `json:"generate_name"`
	Notes        string `json:"notes"`
	Index        int    `json:"index"`
}

type ContainerExecInfoOutput struct {
	HostUri     string `json:"host_uri"`
	PodId       string `json:"pod_id"`
	ContainerId string `json:"container_id"`
}

type ContainerExecInput struct {
	Command []string `json:"command"`
	Tty     bool     `json:"tty"`
}

type ContainerExecSyncInput struct {
	Command []string `json:"command"`
	// Timeout in seconds to stop the command. Default: 0 (run forever).
	Timeout int64 `json:"timeout"`
}

type ContainerExecSyncResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int32  `json:"exit_code"`
}
