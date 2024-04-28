package cadvisor

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/google/cadvisor/cache/memory"
	cadvisormetrics "github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/containerd"
	_ "github.com/google/cadvisor/container/containerd/install"
	"github.com/google/cadvisor/events"
	cadvisorapi "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/utils/sysfs"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	// The amount of time for which to keep stats in memory.
	statsCacheDuration          = 2 * time.Minute
	maxHousekeepingInterval     = 15 * time.Second
	defaultHousekeepingInterval = 10 * time.Second
	allowDynamicHousekeeping    = true
)

func init() {
	ep := "/var/run/onecloud/containerd/containerd.sock"
	containerd.ArgContainerdEndpoint = &ep
	// REF: k8s.io/kubernetes/pkg/kubelet/cadvisor/cadvisor_linux.go
	// override cadvisor flag defaults.
	flagOverrides := map[string]string{
		// Override the default cadvisor housekeeping interval.
		"housekeeping_interval": defaultHousekeepingInterval.String(),
		// Disable event storage by default.
		"event_storage_event_limit": "default=0",
		"event_storage_age_limit":   "default=0",
	}
	for name, defaultValue := range flagOverrides {
		if f := flag.Lookup(name); f != nil {
			f.DefValue = defaultValue
			f.Value.Set(defaultValue)
		} else {
			log.Errorf("Expected cAdvisor flag %q not found", name)
		}
	}
}

type cadvisorClient struct {
	manager.Manager
	rootPath            string
	imageFsInfoProvider ImageFsInfoProvider
}

func New(imageFsInfoProvider ImageFsInfoProvider, rootPath string, cgroupRoots []string) (Interface, error) {
	includedMetrics := cadvisormetrics.MetricSet{
		cadvisormetrics.CpuUsageMetrics:         struct{}{},
		cadvisormetrics.MemoryUsageMetrics:      struct{}{},
		cadvisormetrics.CpuLoadMetrics:          struct{}{},
		cadvisormetrics.DiskIOMetrics:           struct{}{},
		cadvisormetrics.NetworkUsageMetrics:     struct{}{},
		cadvisormetrics.AcceleratorUsageMetrics: struct{}{},
		cadvisormetrics.AppMetrics:              struct{}{},
		cadvisormetrics.ProcessMetrics:          struct{}{},
		cadvisormetrics.DiskUsageMetrics:        struct{}{},
	}
	duration := maxHousekeepingInterval
	allowDynamic := allowDynamicHousekeeping
	housekeepingConfig := manager.HouskeepingConfig{
		Interval:     &duration,
		AllowDynamic: &allowDynamic,
	}

	// Create the cAdvisor container manager
	sysFs := sysfs.NewRealSysFs()
	m, err := manager.New(memory.New(statsCacheDuration, nil), sysFs, housekeepingConfig, includedMetrics, http.DefaultClient, cgroupRoots, "")
	if err != nil {
		return nil, errors.Wrap(err, "new cadvisor manager")
	}

	if _, err := os.Stat(rootPath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path.Clean(rootPath), 0750); err != nil {
				return nil, errors.Wrapf(err, "creating root direcotory %q", rootPath)
			}
		} else {
			return nil, errors.Wrapf(err, "failed to stat %q", rootPath)
		}
	}

	return &cadvisorClient{
		imageFsInfoProvider: imageFsInfoProvider,
		rootPath:            rootPath,
		Manager:             m,
	}, nil
}

func (cc *cadvisorClient) Start() error {
	return cc.Manager.Start()
}

func (cc *cadvisorClient) ContainerInfo(name string, req *cadvisorapi.ContainerInfoRequest) (*cadvisorapi.ContainerInfo, error) {
	return cc.GetContainerInfo(name, req)
}

func (cc *cadvisorClient) ContainerInfoV2(name string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerInfo, error) {
	return cc.GetContainerInfoV2(name, options)
}

func (cc *cadvisorClient) VersionInfo() (*cadvisorapi.VersionInfo, error) {
	return cc.GetVersionInfo()
}

func (cc *cadvisorClient) SubcontainerInfo(name string, req *cadvisorapi.ContainerInfoRequest) (map[string]*cadvisorapi.ContainerInfo, error) {
	infos, err := cc.SubcontainersInfo(name, req)
	if err != nil && len(infos) == 0 {
		return nil, err
	}

	result := make(map[string]*cadvisorapi.ContainerInfo, len(infos))
	for _, info := range infos {
		result[info.Name] = info
	}
	return result, err
}

func (cc *cadvisorClient) MachineInfo() (*cadvisorapi.MachineInfo, error) {
	return cc.GetMachineInfo()
}

func (cc *cadvisorClient) ImagesFsInfo() (cadvisorapiv2.FsInfo, error) {
	label, err := cc.imageFsInfoProvider.ImageFsInfoLabel()
	if err != nil {
		return cadvisorapiv2.FsInfo{}, err
	}
	return cc.getFsInfo(label)
}

func (cc *cadvisorClient) RootFsInfo() (cadvisorapiv2.FsInfo, error) {
	return cc.GetDirFsInfo(cc.rootPath)
}

func (cc *cadvisorClient) getFsInfo(label string) (cadvisorapiv2.FsInfo, error) {
	res, err := cc.GetFsInfo(label)
	if err != nil {
		return cadvisorapiv2.FsInfo{}, err
	}
	if len(res) == 0 {
		return cadvisorapiv2.FsInfo{}, fmt.Errorf("failed to find information for the filesystem labeled %q", label)
	}
	// TODO(vmarmol): Handle this better when a label has more than one image filesystem.
	if len(res) > 1 {
		log.Warningf("More than one filesystem labeled %q: %#v. Only using the first one", label, res)
	}

	return res[0], nil
}

func (cc *cadvisorClient) WatchEvents(request *events.Request) (*events.EventChannel, error) {
	return cc.WatchForEvents(request)
}
