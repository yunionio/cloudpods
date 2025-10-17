package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DifyModelListOptions struct {
	options.BaseListOptions
}

func (o *DifyModelListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type DifyModelShowOptions struct {
	options.BaseShowOptions
}

func (o *DifyModelShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type DifyModelCreateOptions struct {
	apis.SharableVirtualResourceCreateInput

	CPU                    int
	MEMORY                 int    `help:"memory size MB"`
	DISK_SIZE              int    `help:"disk size MB"`
	POSTGRES_IMAGE_ID      string `json:"postgres_image_id"`
	REDIS_IMAGE_ID         string `json:"redis_image_id"`
	NGINX_IMAGE_ID         string `json:"nginx_image_id"`
	DIFY_API_IMAGE_ID      string `json:"dify_api_image_id"`
	DIFY_PLUGIN_IMAGE_ID   string `json:"dify_plugin_image_id"`
	DIFY_WEB_IMAGE_ID      string `json:"dify_web_image_id"`
	DIFY_SANDBOX_IMAGE_ID  string `json:"dify_sandbox_image_id"`
	DIFY_SSRF_IMAGE_ID     string `json:"dify_ssrf_image_id"`
	DIFY_WEAVIATE_IMAGE_ID string `json:"dify_weaviate_image_id"`
	Bandwidth              int
	StorageType            string
	// DiskOverlay    string `help:"disk overlay, e.g. /opt/steam-data/base:/opt/steam-data/games"`
	TemplateId   string
	PortMappings []string `help:"port mapping in the format of protocol:port[:prefix][:first_port_offset][:env_key=env_value], e.g. tcp:5555:192.168.0.0/16:5:WOLF_BASE_PORT=20000"`
	Devices      []string `help:"device info in the format of model[:path[:dev_type]], e.g. 'GeForce RTX 4060'"`

	Env      []string `help:"env in format of key=value"`
	Property []string `help:"extra properties of key=value, e.g. tango32=true"`

	// MountedApps []string `help:"mounted apps, e.g. com.tencent.tmgp.sgame/1.0.0"`

	Entrypoint string `help:"entrypoint"`
}

func (o *DifyModelCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)
	vol := api.Volume{
		SizeMB:      o.DISK_SIZE,
		TemplateId:  o.TemplateId,
		StorageType: o.StorageType,
		// Containers: map[string]*api.ContainerVolumeRelation{
		// 	"1": &api.ContainerVolumeRelation{
		// 		MountPath:    "/etc/wolf",
		// 		SubDirectory: "wolf",
		// 	},
		// 	"2": &api.ContainerVolumeRelation{
		// 		MountPath:    "/home/retro",
		// 		SubDirectory: "home",
		// 	},
		// },
	}
	// if o.DiskOverlay != "" {
	// 	vol.Containers["2"].Overlay = &apis.ContainerVolumeMountDiskOverlay{
	// 		LowerDir: strings.Split(o.DiskOverlay, ":"),
	// 	}
	// }
	vols := []api.Volume{vol}
	dict.Set("volumes", jsonutils.Marshal(vols))
	fetchPortmappings(o.PortMappings, dict)
	fetchDevices(o.Devices, dict)
	fetchEnvs(o.Env, dict)
	fetchProperties(o.Property, dict)
	// fetchMountedApps(o.MountedApps, dict)
	return dict, nil
}

type DifyModelDeleteOptions struct {
	options.BaseIdOptions
}

func (o *DifyModelDeleteOptions) GetId() string {
	return o.ID
}

func (o *DifyModelDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

/* type LLMModelUpdateOptions struct {
	apis.SharableVirtualResourceBaseUpdateInput
	ID string

	Cpu          *int
	Memory       *int `help:"memory size GB"`
	DiskSize     *int `help:"disk size MB"`
	StorageType  string
	TemplateId   string
	NoTemplate   bool `json:"-" help:"remove template"`
	ImageId      string
	Bandwidth    *int
	Dpi          *int
	Fps          *int
	PortMappings []string `help:"port mapping in the format of protocol:port[:prefix][:first_port_offset], e.g. tcp:5555:192.168.0.0/16,10.10.0.0/16:1000"`
	Devices      []string `help:"device info in the format of model[:path[:dev_type]], e.g. QuadraT2A:/dev/nvme1n1, Device::VASTAITECH_GPU"`
	Env          []string `help:"env in the format of key=value, e.g. AUTHENTICATION_PATH=/bupt-test/"`
	Property     []string `help:"extra properties of key=value, e.g. tango32=true"`

	// MountedApps []string `help:"mounted apps, e.g. com.tencent.tmgp.sgame/1.0.0"`

	SyncImage      bool `help:"request sync image" json:"-"`
	ClearSyncImage bool `help:"request clear sync image flag" json:"-"`

	Entrypoint string `help:"entrypoint"`
}

func (o *LLMModelUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMModelUpdateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)
	if o.NoTemplate {
		dict.Set("template_id", jsonutils.NewString(""))
	}
	if o.SyncImage {
		dict.Set("request_sync_image", jsonutils.JSONTrue)
	} else if o.ClearSyncImage {
		dict.Set("request_sync_image", jsonutils.JSONFalse)
	}
	if o.DiskSize != nil && *o.DiskSize > 0 {
		dict.Set("disk_size_mb", jsonutils.NewInt(int64(*o.DiskSize)))
	}

	fetchPortmappings(o.PortMappings, dict)
	fetchDevices(o.Devices, dict)
	fetchEnvs(o.Env, dict)
	fetchProperties(o.Property, dict)
	// fetchMountedApps(o.MountedApps, dict)
	return dict, nil
} */
