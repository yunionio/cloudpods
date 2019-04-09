package zstack

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SHost struct {
	zone *SZone

	Username                string `json:"username"`
	SSHPort                 int    `json:"sshPort"`
	ZoneUUID                string `json:"zoneUuid"`
	Name                    string `json:"name"`
	UUID                    string `json:"uuid"`
	ClusterUUID             string `json:"clusterUuid"`
	Description             string `json:"description"`
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
	//CreateDate              time.Time `json:"createDate"`
	//LastOpDate              time.Time `json:"lastOpDate"`
}

func (host *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (host *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return host.zone.GetIWires()
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := host.zone.getStorages(host.ClusterUUID)
	if err != nil {
		return nil, err
	}
	istorages := []cloudprovider.ICloudStorage{}
	for i := 0; i < len(storages); i++ {
		istorages = append(istorages, &storages[i])
	}
	return istorages, nil
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotFound
}

func (host *SHost) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotFound
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
	//TODO
	return models.HOST_STATUS_RUNNING
}

func (host *SHost) Refresh() error {
	return nil
}

func (host *SHost) GetHostStatus() string {
	return models.HOST_ONLINE
}

func (host *SHost) GetEnabled() bool {
	return true
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

func (host *SHost) GetCpuCount() int8 {
	return int8(host.CPUNum)
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
	return 0
}

func (host *SHost) GetStorageSizeMB() int {
	return 0
}

func (host *SHost) GetStorageType() string {
	return models.DISK_TYPE_HYBRID
}

func (host *SHost) GetHostType() string {
	return models.HOST_TYPE_ZSTACK
}

func (host *SHost) GetManagerId() string {
	return host.zone.region.client.providerID
}

// func (host *SHost) GetInstanceById(instanceId string) (*SInstance, error) {
// 	return nil, cloudprovider.ErrNotImplemented
// }

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
