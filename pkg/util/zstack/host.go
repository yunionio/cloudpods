package zstack

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SHost struct {
	zone *SZone

	ZStackBasic
	Username                string `json:"username"`
	SSHPort                 int    `json:"sshPort"`
	ZoneUUID                string `json:"zoneUuid"`
	ClusterUUID             string `json:"clusterUuid"`
	ManagementIP            string `json:"managementIp"`
	HypervisorType          string `json:"hypervisorType"`
	State                   string `json:"state"`
	Status                  string `json:"status"`
	TotalCPUCapacity        int    `json:"totalCpuCapacity"`
	AvailableCPUCapacity    int    `json:"availableCpuCapacity"`
	CPUSockets              int    `json:"cpuSockets"`
	TotalMemoryCapacity     int    `json:"totalMemoryCapacity"`
	AvailableMemoryCapacity int    `json:"availableMemoryCapacity"`
	CPUNum                  int    `json:"cpuNum"`
	ZStackTime
}

func (host *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (host *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	wires, err := host.zone.region.GetWires(host.ZoneUUID, "", host.ClusterUUID)
	if err != nil {
		return nil, err
	}
	iwires := []cloudprovider.ICloudWire{}
	for i := 0; i < len(wires); i++ {
		iwires = append(iwires, &wires[i])
	}
	return iwires, nil
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	primaryStorages, err := host.zone.region.GetPrimaryStorages(host.zone.UUID, host.ClusterUUID, "")
	if err != nil {
		return nil, err
	}
	istorages := []cloudprovider.ICloudStorage{}
	for i := 0; i < len(primaryStorages); i++ {
		switch primaryStorages[i].Type {
		case StorageTypeLocal:
			storages, err := host.zone.region.getILocalStorages(host.zone, primaryStorages[i].UUID, host.UUID)
			if err != nil {
				return nil, err
			}
			istorages = append(istorages, storages...)
		case StorageTypeCeph:
			storages, err := host.zone.region.getICephStorages(host.zone, primaryStorages[i].UUID)
			if err != nil {
				return nil, err
			}
			istorages = append(istorages, storages...)
		case StorageTypeVCenter:
		}
	}
	return istorages, nil
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.UUID, "", "")
	if err != nil {
		return nil, err
	}
	iInstnace := []cloudprovider.ICloudVM{}
	for i := 0; i < len(instances); i++ {
		instances[i].host = host
		iInstnace = append(iInstnace, &instances[i])
	}
	return iInstnace, nil
}

func (host *SHost) GetIVMById(instanceId string) (cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.UUID, instanceId, "")
	if err != nil {
		return nil, err
	}
	if len(instances) == 1 {
		if instances[0].UUID == instanceId {
			instances[0].host = host
			return &instances[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(instances) == 0 || len(instanceId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (host *SHost) GetId() string {
	return host.UUID
}

func (host *SHost) GetName() string {
	return host.Name
}

func (host *SHost) GetGlobalId() string {
	return host.GetId()
}

func (host *SHost) IsEmulated() bool {
	return false
}

func (host *SHost) GetStatus() string {
	if host.Status == "Connected" {
		return api.HOST_STATUS_RUNNING
	}
	return api.HOST_STATUS_UNKNOWN
}

func (host *SHost) Refresh() error {
	return nil
}

func (host *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (host *SHost) GetEnabled() bool {
	return host.State == "Enabled"
}

func (host *SHost) GetAccessIp() string {
	return host.ManagementIP
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_ZSTACK), "manufacture")
	return info
}

func (host *SHost) GetSN() string {
	return ""
}

func (host *SHost) GetCpuCount() int {
	return host.TotalCPUCapacity
}

func (host *SHost) GetNodeCount() int8 {
	return int8(host.CPUSockets)
}

func (host *SHost) GetCpuDesc() string {
	return ""
}

func (host *SHost) GetCpuMhz() int {
	return 0
}

func (host *SHost) GetMemSizeMB() int {
	return host.TotalMemoryCapacity / 1024 / 1024
}

func (host *SHost) GetStorageSizeMB() int {
	storages, err := host.zone.region.GetPrimaryStorages(host.zone.UUID, host.ClusterUUID, "")
	if err != nil {
		return 0
	}
	totalStorage := 0
	for _, storage := range storages {
		if storage.Type == StorageTypeLocal {
			localStorages, err := host.zone.region.GetLocalStorages(storage.UUID, host.UUID)
			if err != nil {
				return 0
			}
			for i := 0; i < len(localStorages); i++ {
				totalStorage += int(localStorages[i].TotalCapacity)
			}
		}
	}
	return totalStorage / 1024 / 1024
}

func (host *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_ZSTACK
}

func (host *SHost) GetManagerId() string {
	return host.zone.region.client.providerID
}

func (host *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) GetVersion() string {
	return ""
}
