package volcengine

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
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

func (host *SHost) GetStorageSizeMB() int64 {
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
	vms, err := host.zone.region.GetInstances(host.zone.ZoneId, nil)
	if err != nil {
		return nil, err
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		vms[i].host = host
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (host *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := host.zone.region.GetInstance(id)
	if err != nil {
		return nil, err
	}
	vm.host = host
	return vm, nil
}

func (host *SHost) GetInstanceById(instanceId string) (*SInstance, error) {
	inst, err := host.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	inst.host = host
	return inst, nil
}

func (host *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vmId, err := host._createVM(opts)
	if err != nil {
		return nil, err
	}
	vm, err := host.GetInstanceById(vmId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstanceById")
	}
	return vm, nil
}

func (host *SHost) _createVM(opts *cloudprovider.SManagedVMCreateConfig) (string, error) {
	var err error
	if len(opts.PublicKey) > 0 {
		opts.KeypairName, err = host.zone.region.syncKeypair(opts.PublicKey)
		if err != nil {
			return "", err
		}
	}

	if len(opts.InstanceType) == 0 {
		return "", fmt.Errorf("instance type must be specified")
	}
	vmId, err := host.zone.region.CreateInstance(host.zone.ZoneId, opts)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to create %s", opts.InstanceType)
	}
	return vmId, nil
}
