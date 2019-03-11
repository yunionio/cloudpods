package openstack

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type CpuInfo struct {
	Arch     string
	Model    string
	Vendor   string
	Feature  []string
	Topology map[string]int
}

type Service struct {
	Host           string
	ID             string
	DisabledReason string
}

type SResource struct {
	CPU      int8
	DiskGB   int
	Host     string
	MemoryMb int
	Project  string
}

type SHost struct {
	zone *SZone

	CpuInfo string

	CurrentWorkload    int
	Status             string
	State              string
	DiskAvailableLeast int
	HostIP             string
	FreeDiskGB         int
	FreeRamMB          int
	HypervisorHostname string
	HypervisorType     string
	HypervisorVersion  string
	ID                 string
	LocalGB            int
	LocalGbUsed        int
	MemoryMB           int
	MemoryMbUsed       int
	RunningVms         int
	Service            Service
	Vcpus              int8
	VcpusUsed          int8

	// less then version 2.28
	HostName string
	Zone     string
	Resource []map[string]SResource
}

func (host *SHost) GetId() string {
	if len(host.ID) > 0 {
		return host.ID
	}
	return host.HostName
}

func (host *SHost) GetName() string {
	if len(host.Service.Host) > 0 {
		return host.Service.Host
	}
	return host.HostName
}

func (host *SHost) GetGlobalId() string {
	return host.GetId()
}

func (host *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (host *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return host.zone.GetIWires()
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorages()
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.GetName())
	if err != nil {
		return nil, err
	}
	iVMs := []cloudprovider.ICloudVM{}
	for i := 0; i < len(instances); i++ {
		instances[i].host = host
		iVMs = append(iVMs, &instances[i])
	}
	return iVMs, nil
}

func (host *SHost) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	instance, err := host.zone.region.GetInstance(gid)
	if err != nil {
		return nil, err
	}
	instance.host = host
	return instance, nil
}

func (host *SHost) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int,
	vswitchId string, ipAddr string, desc string, passwd string,
	storageType string, diskSizes []int, publicKey string, secgroupId string, userData string,
	bc *billing.SBillingCycle) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHost) CreateVM2(name string, imgId string, sysDiskSize int, instanceType string,
	vswitchId string, ipAddr string, desc string, passwd string,
	storageType string, diskSizes []int, publicKey string, secgroupId string,
	userData string, bc *billing.SBillingCycle) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHost) GetEnabled() bool {
	return true
}

func (host *SHost) GetAccessIp() string {
	return host.HostIP
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_OPENSTACK), "manufacture")
	return info
}

func (host *SHost) GetSN() string {
	return ""
}

func (host *SHost) GetCpuCount() int8 {
	if host.Vcpus > 0 {
		return host.Vcpus
	}
	host.Refresh()
	return host.Vcpus
}

func (host *SHost) GetNodeCount() int8 {
	return host.GetCpuCount()
}

func (host *SHost) GetCpuDesc() string {
	return host.CpuInfo
}

func (host *SHost) GetCpuMhz() int {
	return 0
}

func (host *SHost) GetMemSizeMB() int {
	if host.MemoryMB > 0 {
		return host.MemoryMB
	}
	host.Refresh()
	return host.MemoryMB
}

func (host *SHost) GetStorageSizeMB() int {
	if host.LocalGB > 0 {
		return host.LocalGB * 1024
	}
	host.Refresh()
	return host.LocalGB * 1024
}

func (host *SHost) GetStorageType() string {
	return models.DISK_TYPE_HYBRID
}

func (host *SHost) GetHostType() string {
	return models.HOST_TYPE_OPENSTACK
}

func (host *SHost) GetHostStatus() string {
	switch host.State {
	case "up", "":
		return models.HOST_ONLINE
	default:
		return models.HOST_OFFLINE
	}
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (host *SHost) GetIsMaintenance() bool {
	switch host.Status {
	case "enabled", "":
		return false
	default:
		return true
	}
}

func (host *SHost) GetVersion() string {
	_, version, _ := host.zone.region.GetVersion("compute")
	return version
}

func (host *SHost) GetManagerId() string {
	return host.zone.region.client.providerID
}

func (host *SHost) GetStatus() string {
	return models.HOST_STATUS_RUNNING
}

func (host *SHost) IsEmulated() bool {
	return false
}

func (host *SHost) Refresh() error {
	new, err := host.zone.region.GetIHostById(host.GetId())
	if err != nil {
		return err
	}
	if err := jsonutils.Update(host, new); err != nil {
		return err
	}
	if len(host.Resource) > 0 {
		for _, resouce := range host.Resource {
			for _, info := range resouce {
				if info.Project == "(total)" {
					host.LocalGB = info.DiskGB
					host.Vcpus = info.CPU
					host.MemoryMB = info.MemoryMb
				}
			}
		}
	}
	return nil
}
