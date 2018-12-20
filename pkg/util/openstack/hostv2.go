package openstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SResource struct {
	CPU      int
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
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV2) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorages()
}

func (host *SHostV2) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHostV2) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV2) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV2) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int,
	vswitchId string, ipAddr string, desc string, passwd string,
	storageType string, diskSizes []int, publicKey string, secgroupId string, userData string,
	bc *billing.SBillingCycle) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHostV2) CreateVM2(name string, imgId string, sysDiskSize int, instanceType string,
	vswitchId string, ipAddr string, desc string, passwd string,
	storageType string, diskSizes []int, publicKey string, secgroupId string,
	userData string, bc *billing.SBillingCycle) (cloudprovider.ICloudVM, error) {
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
	return 0
}

func (host *SHostV2) GetNodeCount() int8 {
	return 0
}

func (host *SHostV2) GetCpuDesc() string {
	return ""
}

func (host *SHostV2) GetCpuMhz() int {
	return 0
}

func (host *SHostV2) GetMemSizeMB() int {
	return 0
}

func (host *SHostV2) GetStorageSizeMB() int {
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
	if host.Resource == nil || len(host.Resource) == 0 {
		new, err := host.zone.GetIHostById(host.HostName)
		if err != nil {
			return err
		}
		return jsonutils.Update(host, new)
	}
	return nil
}
