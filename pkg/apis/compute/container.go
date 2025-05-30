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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/sets"

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
	CONTAINER_DEV_NVIDIA_GPU_SHARE = "NVIDIA_GPU_SHARE"
	CONTAINER_DEV_ASCEND_NPU       = "ASCEND_NPU"
	CONTAINER_DEV_VASTAITECH_GPU   = "VASTAITECH_GPU"
)

var (
	CONTAINER_GPU_TYPES = []string{
		CONTAINER_DEV_CPH_AMD_GPU,
		CONTAINER_DEV_NVIDIA_GPU,
		CONTAINER_DEV_NVIDIA_MPS,
		CONTAINER_DEV_NVIDIA_GPU_SHARE,
		CONTAINER_DEV_VASTAITECH_GPU,
	}
)

var NVIDIA_GPU_TYPES = []string{
	CONTAINER_DEV_NVIDIA_GPU,
	CONTAINER_DEV_NVIDIA_MPS,
	CONTAINER_DEV_NVIDIA_GPU_SHARE,
}

const (
	CONTAINER_STORAGE_LOCAL_RAW = "local_raw"
)

const (
	CONTAINER_STATUS_PULLING_IMAGE       = "pulling_image"
	CONTAINER_STATUS_PULL_IMAGE_FAILED   = "pull_image_failed"
	CONTAINER_STATUS_PULLED_IMAGE        = "pulled_image"
	CONTAINER_STATUS_CREATING            = "creating"
	CONTAINER_STATUS_CREATE_FAILED       = "create_failed"
	CONTAINER_STATUS_SAVING_IMAGE        = "saving_image"
	CONTAINER_STATUS_SAVE_IMAGE_FAILED   = "save_image_failed"
	CONTAINER_STATUS_STARTING            = "starting"
	CONTAINER_STATUS_START_FAILED        = "start_failed"
	CONTAINER_STATUS_STOPPING            = "stopping"
	CONTAINER_STATUS_STOP_FAILED         = "stop_failed"
	CONTAINER_STATUS_SYNC_STATUS         = "sync_status"
	CONTAINER_STATUS_SYNC_STATUS_FAILED  = "sync_status_failed"
	CONTAINER_STATUS_UNKNOWN             = "unknown"
	CONTAINER_STATUS_CREATED             = "created"
	CONTAINER_STATUS_EXITED              = "exited"
	CONTAINER_STATUS_CRASH_LOOP_BACK_OFF = "crash_loop_back_off"
	CONTAINER_STATUS_RUNNING             = "running"
	CONTAINER_STATUS_DELETING            = "deleting"
	CONTAINER_STATUS_DELETE_FAILED       = "delete_failed"
	CONTAINER_STATUS_COMMITTING          = "committing"
	CONTAINER_STATUS_COMMIT_FAILED       = "commit_failed"
	// for health check
	CONTAINER_STATUS_PROBING      = "probing"
	CONTAINER_STATUS_PROBE_FAILED = "probe_failed"
	CONTAINER_STATUS_NET_FAILED   = "net_failed"
	// post overlay
	CONTAINER_STATUS_ADD_POST_OVERLY           = "adding_post_overly"
	CONTAINER_STATUS_ADD_POST_OVERLY_FAILED    = "add_post_overly_failed"
	CONTAINER_STATUS_REMOVE_POST_OVERLY        = "removing_post_overly"
	CONTAINER_STATUS_REMOVE_POST_OVERLY_FAILED = "remove_post_overly_failed"
	CONTAINER_STATUS_CACHE_IMAGE               = "caching_image"
	CONTAINER_STATUS_CACHE_IMAGE_FAILED        = "caching_image_failed"
)

var (
	ContainerRunningStatus = sets.NewString(
		CONTAINER_STATUS_RUNNING,
		CONTAINER_STATUS_PROBING,
		CONTAINER_STATUS_PROBE_FAILED,
		CONTAINER_STATUS_NET_FAILED,
	)
	ContainerNoFailedRunningStatus = sets.NewString(CONTAINER_STATUS_RUNNING, CONTAINER_STATUS_PROBING)
	ContainerExitedStatus          = sets.NewString(
		CONTAINER_STATUS_EXITED,
		CONTAINER_STATUS_CRASH_LOOP_BACK_OFF,
	)
	ContainerFinalStatus = sets.NewString(
		CONTAINER_STATUS_RUNNING,
		CONTAINER_STATUS_PROBING,
		CONTAINER_STATUS_PROBE_FAILED,
		CONTAINER_STATUS_NET_FAILED,
		CONTAINER_STATUS_EXITED,
		CONTAINER_STATUS_CRASH_LOOP_BACK_OFF,
	)
)

const (
	CONTAINER_METADATA_CRI_ID           = "cri_id"
	CONTAINER_METADATA_RELEASED_DEVICES = "released_devices"
)

type ContainerSpec struct {
	apis.ContainerSpec
	// Volume mounts
	RootFs       *apis.ContainerRootfs        `json:"rootfs"`
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
	HostId  string `json:"host_id"`
}

type ContainerStopInput struct {
	Timeout int  `json:"timeout"`
	Force   bool `json:"force"`
}

type ContainerSyncStatusResponse struct {
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
	RestartCount int       `json:"restart_count"`
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
	Index   *int                                   `json:"index"`
	Id      string                                 `json:"id"`
	OnlyEnv []*apis.ContainerIsolatedDeviceOnlyEnv `json:"only_env"`
}

type ContainerDevice struct {
	Type           apis.ContainerDeviceType `json:"type"`
	IsolatedDevice *ContainerIsolatedDevice `json:"isolated_device"`
	Host           *ContainerHostDevice     `json:"host"`
}

type ContainerSaveVolumeMountToImageInput struct {
	Name              string   `json:"name"`
	GenerateName      string   `json:"generate_name"`
	Notes             string   `json:"notes"`
	Index             int      `json:"index"`
	Dirs              []string `json:"dirs"`
	UsedByPostOverlay bool     `json:"used_by_post_overlay"`

	DirPrefix string `json:"dir_prefix"`
}

type ContainerExecInfoOutput struct {
	HostUri     string `json:"host_uri"`
	PodId       string `json:"pod_id"`
	ContainerId string `json:"container_id"`
}

type ContainerExecInput struct {
	Command []string `json:"command"`
	Tty     bool     `json:"tty"`
	SetIO   bool     `json:"set_io"`
	Stdin   bool     `json:"stdin"`
	Stdout  bool     `json:"stdout"`
}

type ContainerExecSyncInput struct {
	Command []string `json:"command"`
	// Timeout in seconds to stop the command, 0 mean run forever.
	// default: 0
	Timeout int64 `json:"timeout"`
}

type ContainerExecSyncResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int32  `json:"exit_code"`
}

type ContainerCommitExternalRegistry struct {
	// e.g.: registry.cn-beijing.aliyuncs.com/yunionio
	Url string `json:"url"`
	// authentication configuration
	Auth *apis.ContainerPullImageAuthConfig `json:"auth"`
}

type ContainerCommitInput struct {
	// Container registry id from kubeserver
	RegistryId       string                           `json:"registry_id"`
	ExternalRegistry *ContainerCommitExternalRegistry `json:"external_registry"`
	// image name
	ImageName string `json:"image_name"`
	// image tag
	Tag string `json:"tag"`
}

type ContainerCommitOutput struct {
	Repository string `json:"repository"`
}

type KubeServerContainerRegistryConfigCommon struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type KubeServerContainerRegistryConfigHarbor struct {
	KubeServerContainerRegistryConfigCommon
}

type KubeServerContainerRegistryConfig struct {
	Type   string                                   `json:"type"`
	Common *KubeServerContainerRegistryConfigCommon `json:"common"`
	Harbor *KubeServerContainerRegistryConfigHarbor `json:"harbor"`
}

type KubeServerContainerRegistryDetails struct {
	Id     string                             `json:"id"`
	Name   string                             `json:"name"`
	Url    string                             `json:"url"`
	Type   string                             `json:"type"`
	Config *KubeServerContainerRegistryConfig `json:"config"`
}

type ContainerPerformStatusInput struct {
	apis.PerformStatusInput
	RestartCount   int        `json:"restart_count"`
	StartedAt      *time.Time `json:"started_at"`
	LastFinishedAt *time.Time `json:"last_finished_at"`
}

type ContainerResourcesSetInput struct {
	apis.ContainerResources
	DisableLimitCheck bool `json:"disable_limit_check"`
}

type ContainerVolumeMountAddPostOverlayInput struct {
	Index       int                                         `json:"index"`
	PostOverlay []*apis.ContainerVolumeMountDiskPostOverlay `json:"post_overlay"`
}

type ContainerVolumeMountRemovePostOverlayInput struct {
	Index       int                                         `json:"index"`
	PostOverlay []*apis.ContainerVolumeMountDiskPostOverlay `json:"post_overlay"`
	UseLazy     bool                                        `json:"use_lazy"`
	ClearLayers bool                                        `json:"clear_layers"`
}

type ContainerCacheImageInput struct {
	DiskId string           `json:"disk_id"`
	Image  *CacheImageInput `json:"image"`
}

type ContainerCacheImagesInput struct {
	Images []*ContainerCacheImageInput `json:"images"`
}

func (i *ContainerCacheImagesInput) isImageExists(diskId string, imgId string) bool {
	for idx := range i.Images {
		img := i.Images[idx]
		if img.DiskId != diskId {
			return false
		}
		if img.Image == nil {
			return false
		}
		if img.Image.ImageId == imgId {
			return true
		}
	}
	return false
}

func (i *ContainerCacheImagesInput) Add(diskId string, imgId string, format string) error {
	if diskId == "" {
		return errors.Errorf("diskId is empty")
	}
	if imgId == "" {
		return errors.Errorf("imageId is empty")
	}
	if !i.isImageExists(diskId, imgId) {
		if i.Images == nil {
			i.Images = []*ContainerCacheImageInput{}
		}
		i.Images = append(i.Images, &ContainerCacheImageInput{
			DiskId: diskId,
			Image: &CacheImageInput{
				ImageId:              imgId,
				Format:               format,
				SkipChecksumIfExists: true,
			},
		})
	}
	return nil
}
