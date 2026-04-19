package llm

import (
	"fmt"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
)

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&HostPath{}), func() gotypes.ISerializable {
		return &HostPath{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&HostPaths{}), func() gotypes.ISerializable {
		return &HostPaths{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&ContainerHostPathRelations{}), func() gotypes.ISerializable {
		return &ContainerHostPathRelations{}
	})
}

type ContainerHostPathRelation struct {
	MountPath   string                         `json:"mount_path"`
	ReadOnly    bool                           `json:"read_only"`
	Propagation apis.ContainerMountPropagation `json:"propagation,omitempty"`
	FsUser      *int64                         `json:"fs_user"`
	FsGroup     *int64                         `json:"fs_group"`
}

// key is string format of integer
type ContainerHostPathRelations map[string]*ContainerHostPathRelation

func (s ContainerHostPathRelations) String() string {
	return jsonutils.Marshal(s).String()
}

func (s ContainerHostPathRelations) IsZero() bool {
	return len(s) == 0
}

type HostPath struct {
	Type             apis.ContainerVolumeMountHostPathType              `json:"type"`
	Path             string                                             `json:"path"`
	AutoCreate       bool                                               `json:"auto_create"`
	AutoCreateConfig *apis.ContainerVolumeMountHostPathAutoCreateConfig `json:"auto_create_config,omitempty"`
	// Container index to mount path relation
	Containers ContainerHostPathRelations `json:"containers"`
}

func (s HostPath) String() string {
	return jsonutils.Marshal(s).String()
}

func (s HostPath) IsZero() bool {
	return len(s.Path) == 0
}

func (s HostPath) GetHostPathByContainer(containerIndex int) *ContainerHostPathRelation {
	if len(s.Containers) == 0 {
		return nil
	}
	key := fmt.Sprintf("%d", containerIndex)
	return s.Containers[key]
}

type HostPaths []HostPath

func (s HostPaths) String() string {
	return jsonutils.Marshal(s).String()
}

func (s HostPaths) IsZero() bool {
	return len(s) == 0
}
