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

type SHostV3 struct {
	zone *SZone

	CpuInfo CpuInfo

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
}

func (host *SHostV3) GetId() string {
	return host.ID
}

func (host *SHostV3) GetName() string {
	return host.Service.Host
}

func (host *SHostV3) GetGlobalId() string {
	return host.ID
}

func (host *SHostV3) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (host *SHostV3) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV3) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorages()
}

func (host *SHostV3) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHostV3) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV3) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV3) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int,
	vswitchId string, ipAddr string, desc string, passwd string,
	storageType string, diskSizes []int, publicKey string, secgroupId string, userData string,
	bc *billing.SBillingCycle) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV3) CreateVM2(name string, imgId string, sysDiskSize int, instanceType string,
	vswitchId string, ipAddr string, desc string, passwd string,
	storageType string, diskSizes []int, publicKey string, secgroupId string,
	userData string, bc *billing.SBillingCycle) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV3) GetEnabled() bool {
	return true
}

func (host *SHostV3) GetAccessIp() string {
	return ""
}

func (host *SHostV3) GetAccessMac() string {
	return ""
}

func (host *SHostV3) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_OPENSTACK), "manufacture")
	return info
}

func (host *SHostV3) GetSN() string {
	return ""
}

func (host *SHostV3) GetCpuCount() int8 {
	return host.Vcpus
}

func (host *SHostV3) GetNodeCount() int8 {
	return host.Vcpus
}

func (host *SHostV3) GetCpuDesc() string {
	return jsonutils.Marshal(host.CpuInfo).String()
}

func (host *SHostV3) GetCpuMhz() int {
	return 0
}

func (host *SHostV3) GetMemSizeMB() int {
	return host.MemoryMB
}

func (host *SHostV3) GetStorageSizeMB() int {
	return host.LocalGB
}

func (host *SHostV3) GetStorageType() string {
	return models.DISK_TYPE_HYBRID
}

func (host *SHostV3) GetHostType() string {
	return models.HOST_TYPE_OPENSTACK
}

func (host *SHostV3) GetHostStatus() string {
	switch host.State {
	case "up":
		return models.HOST_ONLINE
	default:
		return models.HOST_OFFLINE
	}
}

func (host *SHostV3) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (host *SHostV3) GetIsMaintenance() bool {
	switch host.Status {
	case "enabled":
		return false
	default:
		return true
	}
}

func (host *SHostV3) GetVersion() string {
	_, version, _ := host.zone.region.GetVersion("compute")
	return version
}

func (host *SHostV3) GetManagerId() string {
	return host.zone.region.client.providerID
}

func (host *SHostV3) GetStatus() string {
	return models.HOST_STATUS_RUNNING
}

func (host *SHostV3) IsEmulated() bool {
	return false
}

func (host *SHostV3) Refresh() error {
	return nil
}
