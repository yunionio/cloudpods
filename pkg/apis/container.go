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

package apis

import "yunion.io/x/pkg/util/sets"

type ContainerKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ContainerSpec struct {
	// Image to use.
	Image string `json:"image"`
	// Image pull policy
	ImagePullPolicy ImagePullPolicy `json:"image_pull_policy"`
	// Command to execute (i.e., entrypoint for docker)
	Command []string `json:"command"`
	// Args for the Command (i.e. command for docker)
	Args []string `json:"args"`
	// Current working directory of the command.
	WorkingDir string `json:"working_dir"`
	// List of environment variable to set in the container.
	Envs []*ContainerKeyValue `json:"envs"`
	// Enable lxcfs
	EnableLxcfs bool `json:"enable_lxcfs"`
	// Volume mounts
	VolumeMounts []*ContainerVolumeMount `json:"volume_mounts"`
}

type ImagePullPolicy string

const (
	ImagePullPolicyAlways       = "Always"
	ImagePullPolicyIfNotPresent = "IfNotPresent"
)

type ContainerVolumeMountType string

const (
	CONTAINER_VOLUME_MOUNT_TYPE_DISK      ContainerVolumeMountType = "disk"
	CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH ContainerVolumeMountType = "host_path"
)

type ContainerDeviceType string

const (
	CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE ContainerDeviceType = "isolated_device"
	CONTAINER_DEVICE_TYPE_HOST            ContainerDeviceType = "host"
)

type ContainerMountPropagation string

const (
	// No mount propagation ("private" in Linux terminology).
	MOUNTPROPAGATION_PROPAGATION_PRIVATE ContainerMountPropagation = "private"
	// Mounts get propagated from the host to the container ("rslave" in Linux).
	MOUNTPROPAGATION_PROPAGATION_HOST_TO_CONTAINER ContainerMountPropagation = "rslave"
	// Mounts get propagated from the host to the container and from the
	// container to the host ("rshared" in Linux).
	MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL ContainerMountPropagation = "rshared"
)

var (
	ContainerMountPropagations = sets.NewString(
		string(MOUNTPROPAGATION_PROPAGATION_PRIVATE), string(MOUNTPROPAGATION_PROPAGATION_HOST_TO_CONTAINER), string(MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL))
)

type ContainerVolumeMount struct {
	Type     ContainerVolumeMountType      `json:"type"`
	Disk     *ContainerVolumeMountDisk     `json:"disk"`
	HostPath *ContainerVolumeMountHostPath `json:"host_path"`
	// Mounted read-only if true, read-write otherwise (false or unspecified).
	ReadOnly bool `json:"read_only"`
	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `json:"mount_path"`
	// If set, the mount needs SELinux relabeling.
	SelinuxRelabel bool `json:"selinux_relabel,omitempty"`
	// Requested propagation mode.
	Propagation ContainerMountPropagation `json:"propagation,omitempty"`
}

type ContainerVolumeMountDisk struct {
	Index           *int   `json:"index,omitempty"`
	Id              string `json:"id"`
	SubDirectory    string `json:"sub_directory"`
	StorageSizeFile string `json:"storage_size_file"`
}

type ContainerVolumeMountHostPath struct {
	Path string `json:"path"`
}
