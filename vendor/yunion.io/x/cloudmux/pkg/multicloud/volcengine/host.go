package volcengine

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
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

func (host *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Id, host.zone.GetId())
}

func (host *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Name, host.zone.GetId())
}

func (host *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Id, host.zone.GetId())
}

func (host *SHost) IsEmulated() bool {
	return true
}

func (host *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (host *SHost) Refresh() error {
	return nil
}

func (host *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (host *SHost) GetEnabled() bool {
	return true
}

func (host *SHost) GetAccessIp() string {
	return ""
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_VOLCENGINE), "manufacture")
	return info
}

func (host *SHost) GetSN() string {
	return ""
}

func (host *SHost) GetCpuCount() int {
	return 0
}

func (host *SHost) GetNodeCount() int8 {
	return 0
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
	return api.DISK_TYPE_HYBRID
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_VOLCENGINE
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) GetVersion() string {
	return VOLCENGINE_API_VERSION
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms := make([]SInstance, 0)
	token := ""
	for {
		parts, nextToken, err := host.zone.region.GetInstances(host.zone.ZoneId, nil, 10, token)
		if err != nil {
			return nil, err
		}
		vms = append(vms, parts...)
		if len(nextToken) == 0 {
			break
		}
		token = nextToken
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		vms[i].host = host
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (host *SHost) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	id := gid
	parts, _, err := host.zone.region.GetInstances(host.zone.ZoneId, []string{id}, 1, "")
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(parts) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	parts[0].host = host
	return &parts[0], nil
}

func (host *SHost) GetInstanceById(instanceId string) (*SInstance, error) {
	inst, err := host.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	inst.host = host
	return inst, nil
}

func (host *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vmId, err := host._createVM(desc.Name, desc.Hostname, desc.ExternalImageId, desc.SysDisk, desc.Cpu, desc.MemoryMB,
		desc.InstanceType, desc.ExternalNetworkId, desc.IpAddr, desc.Description, desc.Password,
		desc.DataDisks, desc.PublicKey, desc.ExternalSecgroupId, desc.UserData, desc.BillingCycle,
		desc.ProjectId, desc.Tags, desc.SPublicIpInfo)
	if err != nil {
		return nil, err
	}
	vm, err := host.GetInstanceById(vmId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstanceById")
	}
	return vm, nil
}

func (host *SHost) _createVM(name string, hostname string, imgId string,
	sysDisk cloudprovider.SDiskInfo, cpu int, memMB int, instanceType string,
	networkID string, ipAddr string, desc string, passwd string,
	dataDisks []cloudprovider.SDiskInfo, publicKey string, secgroupId string,
	userData string, bc *billing.SBillingCycle, projectId string,
	tags map[string]string, publicIp cloudprovider.SPublicIpInfo,
) (string, error) {
	var err error
	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = host.zone.region.syncKeypair(publicKey)
		if err != nil {
			return "", err
		}
	}

	img, err := host.zone.region.GetImage(imgId)
	if err != nil {
		return "", errors.Wrapf(err, "GetImage fail")
	}
	if img.Status != ImageStatusAvailable {
		log.Errorf("image %s status %s", imgId, img.Status)
		return "", fmt.Errorf("image not ready")
	}

	disks := make([]SDisk, len(dataDisks)+1)
	disks[0].Size = img.Size
	if sysDisk.SizeGB > 0 && sysDisk.SizeGB > img.Size {
		disks[0].Size = sysDisk.SizeGB
	}
	storage, err := host.zone.getStorageByCategory(sysDisk.StorageType)
	if err != nil {
		return "", fmt.Errorf("storage %s not avaiable: %s", sysDisk.StorageType, err)
	}
	disks[0].VolumeType = storage.storageType

	for i, dataDisk := range dataDisks {
		disks[i+1].Size = dataDisk.SizeGB
		storage, err := host.zone.getStorageByCategory(dataDisk.StorageType)
		if err != nil {
			return "", fmt.Errorf("storage %s not avaiable: %s", dataDisk.StorageType, err)
		}
		disks[i+1].VolumeType = storage.storageType
	}

	_, err = host.zone.region.GetSecurityGroupDetails(secgroupId)
	if err != nil {
		return "", errors.Wrapf(err, "GetSecurityGroup fail")
	}

	if len(instanceType) == 0 {
		return "", fmt.Errorf("instance type must be specified")
	}
	vmId, err := host.zone.region.CreateInstance(name, hostname, imgId, instanceType, secgroupId, host.zone.ZoneId, desc, passwd, disks, networkID, ipAddr, keypair, userData, bc, projectId, tags)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to create %s", instanceType)
	}
	return vmId, nil
}
