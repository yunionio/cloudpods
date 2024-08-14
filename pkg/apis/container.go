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

import (
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
)

type ContainerKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ContainerLifecyleHandlerType string

const (
	ContainerLifecyleHandlerTypeExec ContainerLifecyleHandlerType = "exec"
)

type ContainerLifecyleHandlerExecAction struct {
	Command []string `json:"command"`
}

type ContainerLifecyleHandler struct {
	Type ContainerLifecyleHandlerType        `json:"type"`
	Exec *ContainerLifecyleHandlerExecAction `json:"exec"`
}

type ContainerLifecyle struct {
	PostStart *ContainerLifecyleHandler `json:"post_start"`
}

type ContainerSecurityContext struct {
	RunAsUser  *int64 `json:"run_as_user,omitempty"`
	RunAsGroup *int64 `json:"run_as_group,omitempty"`
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
	EnableLxcfs        bool                      `json:"enable_lxcfs"`
	Capabilities       *ContainerCapability      `json:"capabilities"`
	Privileged         bool                      `json:"privileged"`
	Lifecyle           *ContainerLifecyle        `json:"lifecyle"`
	CgroupDevicesAllow []string                  `json:"cgroup_devices_allow"`
	SimulateCpu        bool                      `json:"simulate_cpu"`
	ShmSizeMB          int                       `json:"shm_size_mb"`
	SecurityContext    *ContainerSecurityContext `json:"security_context,omitempty"`
	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// Cannot be updated.
	//LivenessProbe *ContainerProbe `json:"liveness_probe,omitempty"`
	// StartupProbe indicates that the Pod has successfully initialized.
	// If specified, no other probes are executed until this completes successfully.
	StartupProbe *ContainerProbe `json:"startup_probe,omitempty"`
}

func (c *ContainerSpec) NeedProbe() bool {
	//if c.LivenessProbe != nil {
	//	return true
	//}
	if c.StartupProbe != nil {
		return true
	}
	return false
}

type ContainerCapability struct {
	Add  []string `json:"add"`
	Drop []string `json:"drop"`
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
	CONTAINER_VOLUME_MOUNT_TYPE_TEXT      ContainerVolumeMountType = "text"
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
	Text     *ContainerVolumeMountText     `json:"text"`
	// Mounted read-only if true, read-write otherwise (false or unspecified).
	ReadOnly bool `json:"read_only"`
	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `json:"mount_path"`
	// If set, the mount needs SELinux relabeling.
	SelinuxRelabel bool `json:"selinux_relabel,omitempty"`
	// Requested propagation mode.
	Propagation ContainerMountPropagation `json:"propagation,omitempty"`
	// Owner permissions
	FsUser  *int64 `json:"fs_user,omitempty"`
	FsGroup *int64 `json:"fs_group,omitempty"`
}

type ContainerOverlayDiskImage struct {
	DiskId  string `json:"disk_id"`
	ImageId string `json:"image_id"`
}

type ContainerDiskOverlayType string

const (
	CONTAINER_DISK_OVERLAY_TYPE_DIRECTORY  ContainerDiskOverlayType = "directory"
	CONTAINER_DISK_OVERLAY_TYPE_DISK_IMAGE ContainerDiskOverlayType = "disk_image"
	CONTAINER_DISK_OVERLAY_TYPE_UNKNOWN    ContainerDiskOverlayType = "unknown"
)

type ContainerVolumeMountDiskOverlay struct {
	LowerDir     []string `json:"lower_dir"`
	UseDiskImage bool     `json:"use_disk_image"`
}

func (o ContainerVolumeMountDiskOverlay) GetType() ContainerDiskOverlayType {
	if len(o.LowerDir) != 0 {
		return CONTAINER_DISK_OVERLAY_TYPE_DIRECTORY
	}
	if o.UseDiskImage {
		return CONTAINER_DISK_OVERLAY_TYPE_DISK_IMAGE
	}
	return CONTAINER_DISK_OVERLAY_TYPE_UNKNOWN
}

func (o ContainerVolumeMountDiskOverlay) IsValid() error {
	if o.GetType() == CONTAINER_DISK_OVERLAY_TYPE_UNKNOWN {
		return errors.ErrNotSupported
	}
	return nil
}

type ContainerVolumeMountDisk struct {
	Index           *int                             `json:"index,omitempty"`
	Id              string                           `json:"id"`
	SubDirectory    string                           `json:"sub_directory"`
	StorageSizeFile string                           `json:"storage_size_file"`
	Overlay         *ContainerVolumeMountDiskOverlay `json:"overlay"`
}

type ContainerVolumeMountHostPathType string

const (
	CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY ContainerVolumeMountHostPathType = "directory"
	CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE      ContainerVolumeMountHostPathType = "file"
)

type ContainerVolumeMountHostPath struct {
	Type ContainerVolumeMountHostPathType `json:"type"`
	Path string                           `json:"path"`
}

type ContainerVolumeMountText struct {
	Content string `json:"content"`
}
