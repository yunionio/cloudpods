package llm

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

type LLMSkuBaseCreateOptions struct {
	apis.SharableVirtualResourceCreateInput

	CPU       int
	MEMORY    int `help:"memory size MB"`
	DISK_SIZE int `help:"disk size MB"`

	Bandwidth   int
	StorageType string
	// DiskOverlay    string `help:"disk overlay, e.g. /opt/steam-data/base:/opt/steam-data/games"`
	TemplateId   string
	PortMappings []string `help:"port mapping in the format of protocol:port[:prefix][:first_port_offset][:env_key=env_value], e.g. tcp:5555:192.168.0.0/16:5:WOLF_BASE_PORT=20000"`
	Devices      []string `help:"device info in the format of model[:path[:dev_type]], e.g. 'GeForce RTX 4060'"`

	Env      []string `help:"env in format of key=value"`
	Property []string `help:"extra properties of key=value, e.g. tango32=true"`

	// MountedApps []string `help:"mounted apps, e.g. com.tencent.tmgp.sgame/1.0.0"`

	Entrypoint string `help:"entrypoint"`
}

func (o *LLMSkuBaseCreateOptions) Params(dict *jsonutils.JSONDict) error {
	vol := api.Volume{
		SizeMB:      o.DISK_SIZE,
		TemplateId:  o.TemplateId,
		StorageType: o.StorageType,
	}
	vols := []api.Volume{vol}
	dict.Set("volumes", jsonutils.Marshal(vols))
	fetchPortmappings(o.PortMappings, dict)
	fetchDevices(o.Devices, dict)
	fetchEnvs(o.Env, dict)
	fetchProperties(o.Property, dict)

	return nil
}

type LLMSkuBaseUpdateOptions struct {
	apis.SharableVirtualResourceBaseUpdateInput

	ID string

	Cpu         *int
	Memory      *int `help:"memory size GB"`
	DiskSize    *int `help:"disk size MB"`
	StorageType string
	TemplateId  string
	NoTemplate  bool `json:"-" help:"remove template"`
	Bandwidth   *int
	// Dpi          *int
	// Fps          *int
	PortMappings []string `help:"port mapping in the format of protocol:port[:prefix][:first_port_offset], e.g. tcp:5555:192.168.0.0/16,10.10.0.0/16:1000"`
	Devices      []string `help:"device info in the format of model[:path[:dev_type]], e.g. QuadraT2A:/dev/nvme1n1, Device::VASTAITECH_GPU"`
	Env          []string `help:"env in the format of key=value, e.g. AUTHENTICATION_PATH=/bupt-test/"`
	Property     []string `help:"extra properties of key=value, e.g. tango32=true"`

	// MountedApps []string `help:"mounted apps, e.g. com.tencent.tmgp.sgame/1.0.0"`

	SyncImage      bool `help:"request sync image" json:"-"`
	ClearSyncImage bool `help:"request clear sync image flag" json:"-"`

	Entrypoint string `help:"entrypoint"`
}

func (o *LLMSkuBaseUpdateOptions) Params(dict *jsonutils.JSONDict) error {
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
	return nil
}

func fetchPortmappings(pmStrs []string, dict *jsonutils.JSONDict) {
	pms := make([]api.PortMapping, 0)
	for _, pm := range pmStrs {
		segs := strings.Split(pm, ":")
		if len(segs) > 1 {
			port, _ := strconv.ParseInt(segs[1], 10, 64)
			var remoteIps []string
			if len(segs) > 2 {
				for _, ip := range strings.Split(segs[2], ",") {
					ip = strings.TrimSpace(ip)
					if len(ip) > 0 {
						remoteIps = append(remoteIps, ip)
					}
				}
			}
			firstPortOffset := 0
			var err error
			if len(segs) > 3 {
				firstPortOffset, err = strconv.Atoi(segs[3])
				if err != nil {
					panic(fmt.Sprintf("parse firstPortOffset: %s", err))
				}
			}
			pm := api.PortMapping{
				Protocol:      segs[0],
				ContainerPort: int(port),
				RemoteIps:     remoteIps,
			}
			if firstPortOffset >= 0 {
				pm.FirstPortOffset = &firstPortOffset
			}
			if len(segs) > 4 {
				envs := make([]computeapi.GuestPortMappingEnv, 0)
				for _, env := range strings.Split(segs[4], ",") {
					parts := strings.Split(env, "=")
					if len(parts) != 2 {
						panic(fmt.Sprintf("parse env: %s", env))
					}
					key := parts[0]
					valType := parts[1]
					switch valType {
					case "port":
						envs = append(envs, computeapi.GuestPortMappingEnv{Key: key, ValueFrom: computeapi.GuestPortMappingEnvValueFromPort})
					case "host_port":
						envs = append(envs, computeapi.GuestPortMappingEnv{Key: key, ValueFrom: computeapi.GuestPortMappingEnvValueFromHostPort})
					default:
						panic(fmt.Sprintf("wrong env type: %q", valType))
					}
				}
				pm.Envs = envs
			}

			pms = append(pms, pm)
		}
	}
	if len(pms) > 0 {
		dict.Set("port_mappings", jsonutils.Marshal(pms))
	}
}

func fetchDevices(devStrs []string, dict *jsonutils.JSONDict) {
	devs := make([]api.Device, 0)
	for _, dev := range devStrs {
		segs := strings.Split(dev, ":")
		if len(segs) > 0 {
			devpath := ""
			devType := ""
			if len(segs) > 1 {
				devpath = segs[1]
			}
			if len(segs) > 2 {
				devType = segs[2]
			}
			devs = append(devs, api.Device{
				Model:      segs[0],
				DevicePath: devpath,
				DevType:    devType,
			})
		}
	}
	if len(devs) > 0 {
		dict.Set("devices", jsonutils.Marshal(devs))
	}
}

func fetchEnvs(EnvStrs []string, dict *jsonutils.JSONDict) {
	envs := make(api.Envs, 0)
	for _, env := range EnvStrs {
		pos := strings.Index(env, "=")
		if pos > 0 {
			key := strings.TrimSpace(env[:pos])
			val := strings.TrimSpace(env[pos+1:])
			envs = append(envs, api.Env{Key: key, Value: val})
		}
	}
	if len(envs) > 0 {
		dict.Set("envs", jsonutils.Marshal(envs))
	}
}

func fetchProperties(propStrs []string, dict *jsonutils.JSONDict) {
	props := make(map[string]string, 0)
	for _, env := range propStrs {
		pos := strings.Index(env, "=")
		if pos > 0 {
			key := strings.TrimSpace(env[:pos])
			val := strings.TrimSpace(env[pos+1:])
			props[key] = val
		}
	}
	if len(props) > 0 {
		dict.Set("properties", jsonutils.Marshal(props))
	}
}

func fetchMountedModels(mdls []string, dict *jsonutils.JSONDict) {
	if len(mdls) > 0 {
		dict.Set("mounted_models", jsonutils.Marshal(mdls))
	}
}
