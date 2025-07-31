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

package host

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type ContainerVolumeMountDisk struct {
	Index                *int                                        `json:"index,omitempty"`
	Id                   string                                      `json:"id"`
	TemplateId           string                                      `json:"template_id"`
	SubDirectory         string                                      `json:"sub_directory"`
	StorageSizeFile      string                                      `json:"storage_size_file"`
	Overlay              *apis.ContainerVolumeMountDiskOverlay       `json:"overlay"`
	CaseInsensitivePaths []string                                    `json:"case_insensitive_paths"`
	PostOverlay          []*apis.ContainerVolumeMountDiskPostOverlay `json:"post_overlay"`
	ResGid               int                                         `json:"res_gid"`
	ResUid               int                                         `json:"res_uid"`
}

type ContainerVolumeMountCephFS struct {
	Id      string `json:"id"`
	MonHost string `json:"mon_host"`
	Path    string `json:"path"`
	Secret  string `json:"secret"`
	Name    string `json:"name"`
}

type ContainerRootfs struct {
	Type apis.ContainerVolumeMountType `json:"type"`
	Disk *ContainerVolumeMountDisk     `json:"disk"`
	// CephFS *ContainerVolumeMountCephFS   `json:"ceph_fs"`
}

type ContainerVolumeMount struct {
	// 用于标识当前 pod volume mount 的唯一性
	UniqueName string                             `json:"unique_name"`
	Type       apis.ContainerVolumeMountType      `json:"type"`
	Disk       *ContainerVolumeMountDisk          `json:"disk"`
	HostPath   *apis.ContainerVolumeMountHostPath `json:"host_path"`
	Text       *apis.ContainerVolumeMountText     `json:"text"`
	CephFS     *ContainerVolumeMountCephFS        `json:"ceph_fs"`
	// Mounted read-only if true, read-write otherwise (false or unspecified).
	ReadOnly bool `json:"read_only"`
	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `json:"mount_path"`
	// If set, the mount needs SELinux relabeling.
	SelinuxRelabel bool `json:"selinux_relabel,omitempty"`
	// Requested propagation mode.
	Propagation apis.ContainerMountPropagation `json:"propagation,omitempty"`
	FsUser      *int64                         `json:"fs_user,omitempty"`
	FsGroup     *int64                         `json:"fs_group,omitempty"`
}

type ContainerSpec struct {
	apis.ContainerSpec
	ImageCredentialToken string                  `json:"image_credential_token"`
	Rootfs               *ContainerRootfs        `json:"rootfs"`
	VolumeMounts         []*ContainerVolumeMount `json:"volume_mounts"`
	Devices              []*ContainerDevice      `json:"devices"`
}

type ContainerDevice struct {
	Type           apis.ContainerDeviceType `json:"type"`
	ContainerPath  string                   `json:"container_path"`
	Permissions    string                   `json:"permissions"`
	IsolatedDevice *ContainerIsolatedDevice `json:"isolated_device"`
	Host           *ContainerHostDevice     `json:"host"`
	Disk           *ContainerDiskDevice     `json:"disk"`
}

type ContainerIsolatedDevice struct {
	Id          string                                 `json:"id"`
	Addr        string                                 `json:"addr"`
	Path        string                                 `json:"path"`
	DeviceType  string                                 `json:"device_type"`
	CardPath    string                                 `json:"card_path"`
	RenderPath  string                                 `json:"render_path"`
	Index       int                                    `json:"index"`
	DeviceMinor int                                    `json:"device_minor"`
	OnlyEnv     []*apis.ContainerIsolatedDeviceOnlyEnv `json:"only_env"`
}

type ContainerHostDevice struct {
	// Path of the device on the host.
	HostPath string `json:"host_path"`
}

type ContainerDiskDevice struct {
	Id string `json:"id"`
}

type ContainerCreateInput struct {
	Name         string         `json:"name"`
	GuestId      string         `json:"guest_id"`
	Spec         *ContainerSpec `json:"spec"`
	RestartCount int            `json:"restart_count"`
}

type ContainerPullImageInput struct {
	Image      string                             `json:"image"`
	PullPolicy apis.ImagePullPolicy               `json:"pull_policy"`
	Auth       *apis.ContainerPullImageAuthConfig `json:"auth"`
}

type ContainerPushImageInput struct {
	Image string                             `json:"image"`
	Auth  *apis.ContainerPullImageAuthConfig `json:"auth"`
}

type ContainerDesc struct {
	Id             string         `json:"id"`
	Name           string         `json:"name"`
	Spec           *ContainerSpec `json:"spec"`
	StartedAt      time.Time      `json:"started_at"`
	LastFinishedAt time.Time      `json:"last_finished_at"`
	RestartCount   int            `json:"restart_count"`
}

type ContainerSaveVolumeMountToImageInput struct {
	ImageId string `json:"image_id"`

	VolumeMountIndex int                   `json:"volume_mount_index"`
	VolumeMount      *ContainerVolumeMount `json:"volume_mount"`
	VolumeMountDirs  []string              `json:"volume_mount_dirs"`

	VolumeMountPrefix string `json:"volume_mount_prefix"`
}

type ContainerCommitInput struct {
	Repository string                             `json:"repository"`
	Auth       *apis.ContainerPullImageAuthConfig `json:"auth"`
}

type ContainerStopInput struct {
	Timeout       int64  `json:"timeout"`
	ShmSizeMB     int    `json:"shm_size_mb"`
	ContainerName string `json:"container_name"`
	Force         bool   `json:"force"`
}
