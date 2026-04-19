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
	HostPaths    []string `json:"-" help:"host path mount in format path=<host_path>,type=<directory|file>,container_index=<index>,mount_path=<container_path>[,auto_create=<bool>][,read_only=<bool>][,propagation=<private|rslave|rshared>][,fs_user=<uid>][,fs_group=<gid>][,uid=<uid>][,gid=<gid>][,permissions=<mode>]; repeatable"`

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
	if err := fetchHostPaths(o.HostPaths, dict); err != nil {
		return err
	}
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
	HostPaths    []string `json:"-" help:"host path mount in format path=<host_path>,type=<directory|file>,container_index=<index>,mount_path=<container_path>[,auto_create=<bool>][,read_only=<bool>][,propagation=<private|rslave|rshared>][,fs_user=<uid>][,fs_group=<gid>][,uid=<uid>][,gid=<gid>][,permissions=<mode>]; repeatable"`
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
	if err := fetchHostPaths(o.HostPaths, dict); err != nil {
		return err
	}
	fetchEnvs(o.Env, dict)
	fetchProperties(o.Property, dict)
	// fetchMountedApps(o.MountedApps, dict)
	return nil
}

type hostPathCliSpec struct {
	Path             string
	Type             apis.ContainerVolumeMountHostPathType
	ContainerIndex   int
	MountPath        string
	ReadOnly         bool
	Propagation      apis.ContainerMountPropagation
	FsUser           *int64
	FsGroup          *int64
	AutoCreate       bool
	AutoCreateConfig *apis.ContainerVolumeMountHostPathAutoCreateConfig
}

func fetchHostPaths(hostPathStrs []string, dict *jsonutils.JSONDict) error {
	hostPaths, err := parseHostPaths(hostPathStrs)
	if err != nil {
		return err
	}
	if len(hostPaths) > 0 {
		dict.Set("host_paths", jsonutils.Marshal(hostPaths))
	}
	return nil
}

func parseHostPaths(hostPathStrs []string) (api.HostPaths, error) {
	if len(hostPathStrs) == 0 {
		return nil, nil
	}
	ret := make(api.HostPaths, 0)
	groupIndex := make(map[string]int)
	for _, item := range hostPathStrs {
		spec, err := parseHostPath(item)
		if err != nil {
			return nil, err
		}
		key := getHostPathGroupKey(spec)
		idx, ok := groupIndex[key]
		if !ok {
			ret = append(ret, api.HostPath{
				Type:             spec.Type,
				Path:             spec.Path,
				AutoCreate:       spec.AutoCreate,
				AutoCreateConfig: spec.AutoCreateConfig,
				Containers:       make(api.ContainerHostPathRelations),
			})
			idx = len(ret) - 1
			groupIndex[key] = idx
		}
		containerKey := strconv.Itoa(spec.ContainerIndex)
		if _, exists := ret[idx].Containers[containerKey]; exists {
			return nil, fmt.Errorf("duplicate host_path container_index %d for path %q", spec.ContainerIndex, spec.Path)
		}
		ret[idx].Containers[containerKey] = &api.ContainerHostPathRelation{
			MountPath:   spec.MountPath,
			ReadOnly:    spec.ReadOnly,
			Propagation: spec.Propagation,
			FsUser:      spec.FsUser,
			FsGroup:     spec.FsGroup,
		}
	}
	return ret, nil
}

func parseHostPath(item string) (*hostPathCliSpec, error) {
	parts := strings.Split(item, ",")
	fields := make(map[string]string, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid host path %q, expected key=value", item)
		}
		key := strings.TrimSpace(part[:idx])
		val := strings.TrimSpace(part[idx+1:])
		if key == "" {
			return nil, fmt.Errorf("invalid host path %q, empty key", item)
		}
		fields[key] = val
	}

	spec := &hostPathCliSpec{
		Type: apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
	}
	for key, val := range fields {
		switch key {
		case "path":
			spec.Path = val
		case "type":
			spec.Type = apis.ContainerVolumeMountHostPathType(val)
		case "container_index":
			idx, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid host path container_index %q: %w", val, err)
			}
			spec.ContainerIndex = idx
		case "mount_path":
			spec.MountPath = val
		case "read_only":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("invalid host path read_only %q: %w", val, err)
			}
			spec.ReadOnly = b
		case "auto_create":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return nil, fmt.Errorf("invalid host path auto_create %q: %w", val, err)
			}
			spec.AutoCreate = b
		case "propagation":
			if !apis.ContainerMountPropagations.Has(val) {
				return nil, fmt.Errorf("invalid host path propagation %q", val)
			}
			spec.Propagation = apis.ContainerMountPropagation(val)
		case "fs_user":
			fsUser, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid host path fs_user %q: %w", val, err)
			}
			spec.FsUser = &fsUser
		case "fs_group":
			fsGroup, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid host path fs_group %q: %w", val, err)
			}
			spec.FsGroup = &fsGroup
		case "uid", "gid", "permissions":
			if spec.AutoCreateConfig == nil {
				spec.AutoCreateConfig = &apis.ContainerVolumeMountHostPathAutoCreateConfig{}
			}
			spec.AutoCreate = true
			switch key {
			case "uid":
				uid, err := strconv.ParseUint(val, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid host path uid %q: %w", val, err)
				}
				spec.AutoCreateConfig.Uid = uint(uid)
			case "gid":
				gid, err := strconv.ParseUint(val, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid host path gid %q: %w", val, err)
				}
				spec.AutoCreateConfig.Gid = uint(gid)
			case "permissions":
				spec.AutoCreateConfig.Permissions = val
			}
		default:
			return nil, fmt.Errorf("invalid host path key %q", key)
		}
	}

	if spec.Path == "" {
		return nil, fmt.Errorf("invalid host path %q, missing path", item)
	}
	if spec.MountPath == "" {
		return nil, fmt.Errorf("invalid host path %q, missing mount_path", item)
	}
	if !isValidHostPathType(spec.Type) {
		return nil, fmt.Errorf("invalid host path type %q", spec.Type)
	}
	if _, ok := fields["container_index"]; !ok {
		return nil, fmt.Errorf("invalid host path %q, missing container_index", item)
	}
	return spec, nil
}

func getHostPathGroupKey(spec *hostPathCliSpec) string {
	uid := uint(0)
	gid := uint(0)
	permissions := ""
	if spec.AutoCreateConfig != nil {
		uid = spec.AutoCreateConfig.Uid
		gid = spec.AutoCreateConfig.Gid
		permissions = spec.AutoCreateConfig.Permissions
	}
	return fmt.Sprintf("%s|%s|%t|%d|%d|%s", spec.Path, spec.Type, spec.AutoCreate, uid, gid, permissions)
}

func isValidHostPathType(hostPathType apis.ContainerVolumeMountHostPathType) bool {
	switch hostPathType {
	case apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY, apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE:
		return true
	default:
		return false
	}
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
