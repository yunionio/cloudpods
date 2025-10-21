package llm

import (
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/pkg/gotypes"
)

type HostInfo struct {
	HostId       string
	Host         string
	HostAccessIp string
	HostEIP      string
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&PortMappings{}), func() gotypes.ISerializable {
		return &PortMappings{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&Devices{}), func() gotypes.ISerializable {
		return &Devices{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&PortMappingEnvs{}), func() gotypes.ISerializable {
		return &PortMappingEnvs{}
	})
	gotypes.RegisterSerializable(reflect.TypeOf(&Envs{}), func() gotypes.ISerializable {
		return &Envs{}
	})
}

type PortMappingEnv struct {
	Key       string `json:"key"`
	ValueFrom string `json:"value_from"`
}

type PortMappingEnvs []PortMappingEnv

func (pm PortMappingEnvs) String() string {
	return jsonutils.Marshal(pm).String()
}

func (pm PortMappingEnvs) IsZero() bool {
	return len(pm) == 0
}

type PortMapping struct {
	Protocol        string                           `json:"protocol"`
	ContainerPort   int                              `json:"container_port"`
	RemoteIps       []string                         `json:"remote_ips"`
	FirstPortOffset *int                             `json:"first_port_offset"`
	Envs            []computeapi.GuestPortMappingEnv `json:"envs"`
}

type PortMappings []PortMapping

func (s PortMappings) String() string {
	return jsonutils.Marshal(s).String()
}

func (s PortMappings) IsZero() bool {
	return len(s) == 0
}

type Device struct {
	DevType    string `json:"dev_type"`
	Model      string `json:"model"`
	DevicePath string `json:"device_path"`
}

type Devices []Device

func (s Devices) String() string {
	return jsonutils.Marshal(s).String()
}

func (s Devices) IsZero() bool {
	return len(s) == 0
}

type Env struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Envs []Env

func (s Envs) String() string {
	return jsonutils.Marshal(s).String()
}

func (s Envs) IsZero() bool {
	return len(s) == 0
}

type LLMModelDetails struct {
	apis.SharableVirtualResourceDetails
	// 当前大模型套餐包含的实例个数。
	LLMCapacity int
	Image       string
	ImageLabel  string
	ImageName   string

	Template string
}

// type MountedAppResourceDetails struct {
// 	MountedApps []string `json:"mounted_apps"`
// }

type LLMModelListInput struct {
	apis.SharableVirtualResourceListInput
	// MountedAppResourceListInput

	LLMType string `json:"llm_type"`
}

type LLMModelBaseCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	Cpu    int `json:"cpu"`
	Memory int `json:"memory"`

	Bandwidth int `json:"bandwidth"`

	Volumes      *Volumes          `json:"volumes"`
	PortMappings *PortMappings     `json:"port_mappings"`
	Devices      *Devices          `json:"devices"`
	Envs         *Envs             `json:"envs"`
	Properties   map[string]string `json:"properties"`
}

type LLMModelCreateInput struct {
	LLMModelBaseCreateInput

	LLMImageId   string `json:"llm_image_id"`
	LLMType      string `json:"llm_type"`
	LLMModelName string `json:"llm_model_name"`
}

type LLMModelUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput
	// MountedAppResourceUpdateInput

	LLMImageId   string `json:"llm_image_id"`
	LLMModelName string `json:"llm_model_name"`
	Cpu          *int   `json:"cpu"`
	Memory       *int   `json:"memory"`

	RequestSyncImage *bool    `json:"request_sync_image"`
	DiskSizeMB       *int     `json:"disk_size_mb"`
	TemplateId       *string  `json:"template_id"`
	StorageType      *string  `json:"storage_type"`
	Volumes          *Volumes `json:"volumes"`

	Bandwidth    *int              `json:"bandwidth"`
	PortMappings *PortMappings     `json:"port_mappings"`
	Devices      *Devices          `json:"devices"`
	Envs         *Envs             `json:"envs"`
	Properties   map[string]string `json:"properties"`
}

// type LLMModelCloneInput struct {
// 	Name string `json:"name"`
// }

// type LLMModelSyncImageRequestTaskInput struct {
// 	Request bool `json:"request"`
// }

type DifyModelListInput struct {
	apis.SharableVirtualResourceListInput
	// MountedAppResourceListInput
}

type DifyModelCreateInput struct {
	LLMModelBaseCreateInput

	PostgresImageId     string `json:"postgres_image_id"`
	RedisImageId        string `json:"redis_image_id"`
	NginxImageId        string `json:"nginx_image_id"`
	DifyApiImageId      string `json:"dify_api_image_id"`
	DifyPluginImageId   string `json:"dify_plugin_image_id"`
	DifyWebImageId      string `json:"dify_web_image_id"`
	DifySandboxImageId  string `json:"dify_sandbox_image_id"`
	DifySSRFImageId     string `json:"dify_ssrf_image_id"`
	DifyWeaviateImageId string `json:"dify_weaviate_image_id"`
}
