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
	"encoding/json"

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

type ContainerProcMountType string

const (
	// DefaultProcMount uses the container runtime defaults for readonly and masked
	// paths for /proc.  Most container runtimes mask certain paths in /proc to avoid
	// accidental security exposure of special devices or information.
	ContainerDefaultProcMount ContainerProcMountType = "Default"

	// UnmaskedProcMount bypasses the default masking behavior of the container
	// runtime and ensures the newly created /proc the container stays in tact with
	// no modifications.
	ContainerUnmaskedProcMount ContainerProcMountType = "Unmasked"
)

type ContainerSecurityContext struct {
	RunAsUser  *int64 `json:"run_as_user,omitempty"`
	RunAsGroup *int64 `json:"run_as_group,omitempty"`
	// procMount denotes the type of proc mount to use for the containers.
	// The default is DefaultProcMount which uses the container runtime defaults for
	ProcMount       ContainerProcMountType `json:"proc_mount"`
	ApparmorProfile string                 `json:"apparmor_profile"`
}

type ContainerResources struct {
	// CpuCfsQuota can be set to 0.5 that mapping to 0.5*100000 for cpu.cpu_cfs_quota_us
	CpuCfsQuota *float64 `json:"cpu_cfs_quota,omitempty"`
	// MemoryLimitMB will be transferred to memory.limit_in_bytes
	// MemoryLimitMB *int64 `json:"memory_limit_mb,omitempty"`
	// PidsMax will be set to pids.max
	PidsMax *int `json:"pids_max"`
	// DevicesAllow will be set to devices.allow
	DevicesAllow []string `json:"devices_allow"`
	// This flag only affects the cpuset controller. If the clone_children
	// flag is enabled in a cgroup, a new cpuset cgroup will copy its
	// configuration fromthe parent during initialization.
	CpusetCloneChildren bool `json:"cpuset_clone_children"`
}

type ContainerEnvRefValueType string

const (
	ContainerEnvRefValueTypeIsolatedDevice ContainerEnvRefValueType = "isolated_device"
)

type ContainerIsolatedDeviceOnlyEnv struct {
	Key             string `json:"key"`
	FromRenderPath  bool   `json:"from_render_path"`
	FromIndex       bool   `json:"from_index"`
	FromDeviceMinor bool   `json:"from_device_minor"`
}

type ContainerSpec struct {
	// Image to use.
	Image string `json:"image"`
	// Image pull policy
	ImagePullPolicy ImagePullPolicy `json:"image_pull_policy"`
	// Image credential id
	ImageCredentialId string `json:"image_credential_id"`
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
	DisableNoNewPrivs  bool                      `json:"disable_no_new_privs"`
	Lifecyle           *ContainerLifecyle        `json:"lifecyle"`
	CgroupDevicesAllow []string                  `json:"cgroup_devices_allow"`
	CgroupPidsMax      int                       `json:"cgroup_pids_max"`
	ResourcesLimit     *ContainerResources       `json:"resources_limit"`
	SimulateCpu        bool                      `json:"simulate_cpu"`
	ShmSizeMB          int                       `json:"shm_size_mb"`
	SecurityContext    *ContainerSecurityContext `json:"security_context,omitempty"`
	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// Cannot be updated.
	//LivenessProbe *ContainerProbe `json:"liveness_probe,omitempty"`
	// StartupProbe indicates that the Pod has successfully initialized.
	// If specified, no other probes are executed until this completes successfully.
	StartupProbe  *ContainerProbe `json:"startup_probe,omitempty"`
	AlwaysRestart bool            `json:"always_restart"`
	Primary       bool            `json:"primary"`
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
	CONTAINER_VOLUME_MOUNT_TYPE_CEPHF_FS  ContainerVolumeMountType = "ceph_fs"
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
	// 用于标识当前 pod volume mount 的唯一性
	UniqueName string                        `json:"unique_name"`
	Type       ContainerVolumeMountType      `json:"type"`
	Disk       *ContainerVolumeMountDisk     `json:"disk"`
	HostPath   *ContainerVolumeMountHostPath `json:"host_path"`
	Text       *ContainerVolumeMountText     `json:"text"`
	CephFS     *ContainerVolumeMountCephFS   `json:"ceph_fs"`
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

type ContainerVolumeMountDiskPostImageOverlay struct {
	Id      string            `json:"id"`
	PathMap map[string]string `json:"path_map"`
}

type ContainerVolumeMountDiskPostImageOverlayUnpacker ContainerVolumeMountDiskPostImageOverlay

func (ov *ContainerVolumeMountDiskPostImageOverlay) UnmarshalJSON(data []byte) error {
	nov := new(ContainerVolumeMountDiskPostImageOverlayUnpacker)
	if err := json.Unmarshal(data, nov); err != nil {
		return err
	}
	ov.Id = nov.Id
	// 防止 PathMap 被合并，总是用 Unarmshal data 里面的 path_map
	ov.PathMap = nov.PathMap
	return nil
}

type ContainerVolumeMountDiskPostOverlayType string

const (
	CONTAINER_VOLUME_MOUNT_DISK_POST_OVERLAY_HOSTPATH ContainerVolumeMountDiskPostOverlayType = "host_path"
	CONTAINER_VOLUME_MOUNT_DISK_POST_OVERLAY_IMAGE    ContainerVolumeMountDiskPostOverlayType = "image"
)

type ContainerVolumeMountDiskPostOverlay struct {
	// 宿主机底层目录
	HostLowerDir []string `json:"host_lower_dir"`
	// 合并后要挂载到容器的目录
	ContainerTargetDir string                                    `json:"container_target_dir"`
	Image              *ContainerVolumeMountDiskPostImageOverlay `json:"image"`
	FsUser             *int64                                    `json:"fs_user,omitempty"`
	FsGroup            *int64                                    `json:"fs_group,omitempty"`
}

func (o ContainerVolumeMountDiskPostOverlay) IsEqual(input ContainerVolumeMountDiskPostOverlay) bool {
	if o.GetType() != input.GetType() {
		return false
	}
	switch o.GetType() {
	case CONTAINER_VOLUME_MOUNT_DISK_POST_OVERLAY_HOSTPATH:
		return o.ContainerTargetDir == input.ContainerTargetDir
	case CONTAINER_VOLUME_MOUNT_DISK_POST_OVERLAY_IMAGE:
		return o.Image.Id == input.Image.Id
	}
	return false
}

func (o ContainerVolumeMountDiskPostOverlay) GetType() ContainerVolumeMountDiskPostOverlayType {
	if o.Image != nil {
		return CONTAINER_VOLUME_MOUNT_DISK_POST_OVERLAY_IMAGE
	}
	return CONTAINER_VOLUME_MOUNT_DISK_POST_OVERLAY_HOSTPATH
}

type ContainerVolumeMountDisk struct {
	Index           *int   `json:"index,omitempty"`
	Id              string `json:"id"`
	SubDirectory    string `json:"sub_directory"`
	StorageSizeFile string `json:"storage_size_file"`
	// lower overlay 设置，disk 的 volume 会作为 upper，最终 merged 的目录会传给容器
	Overlay *ContainerVolumeMountDiskOverlay `json:"overlay"`
	// case insensitive feature is incompatible with overlayfs
	CaseInsensitivePaths []string `json:"case_insensitive_paths"`
	// 当 disk volume 挂载完后，需要 overlay 的目录设置
	PostOverlay []*ContainerVolumeMountDiskPostOverlay `json:"post_overlay"`
	// The ext2 filesystem reserves a certain percentage of the available space (by default 5%, see mke2fs(8) and tune2fs(8)). These options determine who can use the reserved blocks. (Roughly: whoever has the specified uid, or belongs to the specified group.)
	ResGid int `json:"res_gid"`
	ResUid int `json:"res_uid"`
}

type ContainerVolumeMountHostPathType string

const (
	CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY ContainerVolumeMountHostPathType = "directory"
	CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE      ContainerVolumeMountHostPathType = "file"
)

type ContainerVolumeMountHostPathAutoCreateConfig struct {
	Uid         uint   `json:"uid"`
	Gid         uint   `json:"gid"`
	Permissions string `json:"permissions"`
}

type ContainerVolumeMountHostPath struct {
	Type             ContainerVolumeMountHostPathType              `json:"type"`
	Path             string                                        `json:"path"`
	AutoCreate       bool                                          `json:"auto_create"`
	AutoCreateConfig *ContainerVolumeMountHostPathAutoCreateConfig `json:"auto_create_config,omitempty"`
}

type ContainerVolumeMountText struct {
	Content string `json:"content"`
}

type ContainerVolumeMountCephFS struct {
	Id string `json:"id"`
}

type ContainerPullImageAuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth,omitempty"`
	ServerAddress string `json:"server_address,omitempty"`
	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string `json:"identity_token,omitempty"`
	// RegistryToken is a bearer token to be sent to a registry
	RegistryToken string `json:"registry_token,omitempty"`
}

type ContainerRootfs struct {
	Type ContainerVolumeMountType  `json:"type"`
	Disk *ContainerVolumeMountDisk `json:"disk"`
	//CephFS *ContainerVolumeMountCephFS `json:"ceph_fs"`
}
