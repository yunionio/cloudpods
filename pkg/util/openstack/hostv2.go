package openstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SResource struct {
	CPU      int8
	DiskGB   int
	Host     string
	MemoryMb int
	Project  string
}

type SHostV2 struct {
	zone *SZone

	HostName string
	Zone     string
	Resource []map[string]SResource
}

func (host *SHostV2) GetId() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.providerID, host.zone.GetId())
}

func (host *SHostV2) GetName() string {
	return host.HostName
}

func (host *SHostV2) GetGlobalId() string {
	return host.HostName
}

func (host *SHostV2) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (host *SHostV2) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return host.zone.GetIWires()
}

func (host *SHostV2) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorages()
}

func (host *SHostV2) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHostV2) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.zone.ZoneName, host.HostName)
	if err != nil {
		return nil, err
	}
	iVMs := []cloudprovider.ICloudVM{}
	for i := 0; i < len(instances); i++ {
		instances[i].hostV2 = host
		iVMs = append(iVMs, &instances[i])
	}
	return iVMs, nil
}

func (host *SHostV2) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	instance, err := host.zone.region.GetInstance(gid)
	if err != nil {
		return nil, err
	}
	instance.hostV2 = host
	return instance, nil
}

func (host *SHostV2) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV2) GetEnabled() bool {
	return true
}

func (host *SHostV2) GetAccessIp() string {
	return ""
}

func (host *SHostV2) GetAccessMac() string {
	return ""
}

func (host *SHostV2) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_OPENSTACK), "manufacture")
	return info
}

func (host *SHostV2) GetSN() string {
	return ""
}

func (host *SHostV2) GetCpuCount() int8 {
	if len(host.Resource) == 0 {
		if err := host.Refresh(); err != nil {
			return 0
		}
	}
	for _, resource := range host.Resource {
		for _, info := range resource {
			if info.Project == "(total)" {
				return info.CPU
			}
		}
	}
	return 0
}

func (host *SHostV2) GetNodeCount() int8 {
	return host.GetCpuCount()
}

func (host *SHostV2) GetCpuDesc() string {
	return ""
}

func (host *SHostV2) GetCpuMhz() int {
	return 0
}

func (host *SHostV2) GetMemSizeMB() int {
	if len(host.Resource) == 0 {
		if err := host.Refresh(); err != nil {
			return 0
		}
	}
	for _, resource := range host.Resource {
		for _, info := range resource {
			if info.Project == "(total)" {
				return info.MemoryMb
			}
		}
	}
	return 0
}

func (host *SHostV2) GetStorageSizeMB() int {
	if len(host.Resource) == 0 {
		if err := host.Refresh(); err != nil {
			return 0
		}
	}
	for _, resource := range host.Resource {
		for _, info := range resource {
			if info.Project == "(total)" {
				return info.DiskGB * 1024
			}
		}
	}
	return 0
}

func (host *SHostV2) GetStorageType() string {
	return models.DISK_TYPE_HYBRID
}

func (host *SHostV2) GetHostType() string {
	return models.HOST_TYPE_OPENSTACK
}

func (host *SHostV2) GetHostStatus() string {
	return models.HOST_ONLINE
}

func (host *SHostV2) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (host *SHostV2) GetIsMaintenance() bool {
	return false
}

func (host *SHostV2) GetVersion() string {
	_, version, _ := host.zone.region.GetVersion("compute")
	return version
}

func (host *SHostV2) GetManagerId() string {
	return host.zone.region.client.providerID
}

func (host *SHostV2) GetStatus() string {
	return models.HOST_STATUS_RUNNING
}

func (host *SHostV2) IsEmulated() bool {
	return false
}

func (host *SHostV2) Refresh() error {
	new, err := host.zone.GetIHostById(host.HostName)
	if err != nil {
		return err
	}
	return jsonutils.Update(host, new)
}
