package llm

import (
	"fmt"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&Volume{}), func() gotypes.ISerializable {
		return &Volume{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&Volumes{}), func() gotypes.ISerializable {
		return &Volumes{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&ContainerVolumeRelations{}), func() gotypes.ISerializable {
		return &ContainerVolumeRelations{}
	})
}

const (
	VOLUME_STATUS_READY = computeapi.DISK_READY

	VOLUME_STATUS_START_SYNC_STATUS  = "start_syncstatus"
	VOLUME_STATUS_SYNC_STATUS        = "syncstatus"
	VOLUME_STATUS_SYNC_STATUS_FAILED = "syncstatus_fail"

	VOLUME_STATUS_START_RESET  = "start_reset"
	VOLUME_STATUS_RESETTING    = "resetting"
	VOLUME_STATUS_RESET_FAILED = "reset_fail"

	VOLUME_STATUS_START_RESIZE  = "start_resize"
	VOLUME_STATUS_RESIZING      = "resizing"
	VOLUME_STATUS_RESIZE_FAILED = "resize_fail"

	VOLUME_STATUS_ASSIGNING = "assigning"
)

type VolumeDetails struct {
	apis.VirtualResourceDetails

	Volume

	Template string

	DesktopId     string
	DesktopName   string
	DesktopStatus string

	Disk string

	StorageId     string
	Storage       string
	StorageStatus string
	StorageHosts  []computeapi.StorageHost

	Hosts []HostInfo

	HostInfo

	InstanceId   string
	InstanceName string
}

type ContainerVolumeRelation struct {
	MountPath    string                                `json:"mount_path"`
	SubDirectory string                                `json:"sub_directory"`
	Overlay      *apis.ContainerVolumeMountDiskOverlay `json:"overlay"`
	FsUser       *int64                                `json:"fs_user"`
	FsGroup      *int64                                `json:"fs_group"`
}

// key is string format of integer
type ContainerVolumeRelations map[string]*ContainerVolumeRelation

func (s ContainerVolumeRelations) String() string {
	return jsonutils.Marshal(s).String()
}

func (s ContainerVolumeRelations) IsZero() bool {
	return len(s) == 0
}

type Volume struct {
	// db.SStandaloneAnonResourceBase
	Id          string
	Name        string
	StorageType string
	TemplateId  string
	SizeMB      int
	// Container index to mount path relation
	Containers ContainerVolumeRelations `json:"containers"`
}

func (s Volume) String() string {
	return jsonutils.Marshal(s).String()
}

func (s Volume) IsZero() bool {
	return s.SizeMB == 0
}

func (s Volume) GetVolumeByContainer(containerIndex int) *ContainerVolumeRelation {
	if len(s.Containers) == 0 {
		return nil
	}
	key := fmt.Sprintf("%d", containerIndex)
	return s.Containers[key]
}

type Volumes []Volume

func (s Volumes) String() string {
	return jsonutils.Marshal(s).String()
}

func (s Volumes) IsZero() bool {
	return len(s) == 0
}

type VolumeCreateInput struct {
	apis.VirtualResourceCreateInput
	ImageId     string
	Size        int
	StorageType string

	PreferHost string
}

type VolumePerformResetInput struct {
	SizeGb int `json:"size_gb"`
}

type VolumePerformResizeInput struct {
	Size string `json:"size"`
}

type VolumeResizeTaskInput struct {
	SizeMB        int    `json:"size_mb"`
	DesktopStatus string `json:"desktop_status"`
}

type VolumeListInput struct {
	apis.VirtualResourceListInput

	Host string `json:"host"`

	Unused *bool `json:"unused"`

	Size string `json:"size"`

	DesktopId string `json:"desktop_id"`
}
