package cucloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SInstance struct {
	multicloud.SInstanceBase
	multicloud.STagBase

	host *SHost

	ExtraSpecs        string
	FlavorGroupTypeCn string
	NetworkName       string
	BigRegionName     string
	CPUUnit           string
	ImageStatus       string
	CloudBrandCN      string
	ServerId          string
	//FloatingIps          []interface{}
	FlavorType string
	RegionUUID string
	FlavorUUID string
	HostUUID   string
	//SuspendHistory       []interface{}
	VirtualMachineIp     string
	CloudId              string
	AzId                 string
	ZoneId               string
	NetworkID            string
	ImageUUID            string
	RAM                  int
	NetCards             []SInstanceNic
	FlavorName           string
	ImageId              string
	ImageName            string
	ResourceArchitecture string
	Volumes              []struct {
		VolumeType       string
		VirtualMachineId string
		IsDefault        string
		VolumeUuid       string
		VolumeName       string
		VolumeId         string
		VolumeStatus     string
		VaultStatus      string
		VolumeSize       string
		SysDiskSource    string
	}
	CPU             int
	FlavorGroupType string
	SecurityGroup   []struct {
		SecurityGroupId  string
		VirtualMachineId string
		NetCardName      string
		NetCardId        string
	}
	ProdcutName        string
	Flavor             string
	AccountId          string
	ImageLabel         string
	ExpirationTime     string
	RegionOssCode      string
	ServerType         string
	OperStatus         string
	SubNetworkName     string
	ProvinceName       string
	ZoneCode           string
	VirtualMachineUUID string
	Status             string
	StatusFlag         string
	HostName           string
	VirtualMachineName string
	CloudRegionId      string
	V4SubNetworkName   string
	AccountName        string
	TagId              []struct {
		Key   string
		Value string
	}
	RegionName        string
	FlavorId          string
	FlavorCode        string
	SiteName          string
	NewNetCards       []SInstanceNic
	NcSecurityStatus  string
	VaultStatus       string
	PaymentType       string
	CloudBrand        string
	InstanceId        string
	ResourceGroupId   string
	ZoneName          string
	SubNetworkId      string
	ProductNo         string
	V4SubNetworkId    string
	CloudRegionName   string
	Architecture      string
	ImageVersion      string
	InNetIP           string
	VirtualMachineId  string
	ResourceGroupName string
	RegionOssName     string
	NatForwardRule    string
	VaultBindStatus   string
	ProvinceCode      string
	ImageOs           string
	UserID            string
	CloudRegionCode   string
	CreateTime        string
	RegionId          string
	CloudName         string
	RAMUnit           string
	VaultBindId       string
	VaultBindType     string
}

/*
{"code":200,"message":"获取云主机实例列表成功","result":{"total":1,"pageSize":100,"currentSystemTime":1753103116761,"list":[{"extraSpecs":"{\"trait:CUSTOM_S1\":\"required\"}","flavorGroupTypeCn":"通用型1代-S1","networkName":"test","bigRegionName":"华北地区","cpuUnit":"核","imageStatus":"available","cloudBrandCN":"联通云-行业云","serverId":"8004166212734550016","floatingIps":[],"flavorType":"通用型","regionUuid":"","flavorUuid":"6f547294-f320-40c3-9cb2-3dd25d4800bf","hostUuid":"0178ca1d-e841-41d5-a535-a74c5047c1ae","suspendHistory":[],"virtualMachineIp":"192.168.23.87","cloudId":"7961122105171574784","azId":"7961158477647380480","zoneId":"7961158857508716544","networkId":"8619623900411052032","imageUuid":"73d31dbe-5483-4629-9988-b8de5171e918","ram":"1","netCards":[{"securityGroupId":"8635578403548631040","fixIpAddress":"192.168.23.87","virtualMachineId":"8635622412987052032","netCardName":"netCard_OBP7","netCardDefault":"true","innerFloatingIp":"100.127.24.142","netCardId":"8635622413214871552","networkId":"8619623900411052032","subNetworkId":"8635575140815560704"}],"flavorName":"1C1G","imageId":"8598144440476790784","imageName":"AnolisOS 8.8 64位(标准版)","resourceArchitecture":"x86_x86","volumes":[{"volumeType":"sysDisk","virtualMachineId":"8635622412987052032","isDefault":"true","volumeUuid":"25cb9cd9-eb5c-41d4-b6bc-f198b572fb07","volumeName":"574dc7d4b5114d19","volumeId":"8635622605270724608","volumeStatus":"running","vaultStatus":"unbind","volumeSize":"50","sysDiskSource":"console"}],"cpu":"1","flavorGroupType":"s1.medium1","securityGroup":[{"securityGroupId":"8635578403548631040","virtualMachineId":"8635622412987052032","netCardName":"netCard_OBP7","netCardId":"8635622413214871552"}],"prodcutName":"弹性计算弹性云主机","flavor":"通用型 1CPU 1GB","accountId":"679094","imageLabel":"3","expirationTime":"2025-07-21 23:59:59","regionOssCode":"cn-langfang-2","serverType":"云服务器ECS(x86架构)","operStatus":"create-success","subNetworkName":"test333","provinceName":"河北省","zoneCode":"cn-langfang-2a","virtualMachineUuid":"4fd14a3b-fd7f-4310-9ec2-7af2e043f709","status":"running","statusFlag":"200","hostName":"ecm0035.a.cn-langfang-2","virtualMachineName":"test-vm","cloudRegionId":"7961132062134697984","v4SubNetworkName":"test333","accountName":"马鸿飞","tagId":[],"regionName":"廊坊骨干云池","flavorId":"7961207537586601984","flavorCode":"cpu1ram1","siteName":"中国","newNetCards":[{"fixIpAddress":"192.168.23.87","virtualMachineId":"8635622412987052032","netCardName":"netCard_OBP7","netCardDefault":"true","innerFloatingIp":"100.127.24.142","netCardId":"8635622413214871552","networkId":"8619623900411052032","subNetworkId":"8635575140815560704"}],"ncSecurityStatus":"bind-success","vaultStatus":"unbind","paymentType":"按量付费","cloudBrand":"wocloud-industry","instanceId":"8635622412987052033","resourceGroupId":"8020935097949384704","zoneName":"通用专区1","subNetworkId":"8635575140815560704","productNo":"100-1001","v4SubNetworkId":"8635575140815560704","cloudRegionName":"廊坊二区","architecture":"x86","imageVersion":"8.8","inNetIp":"172.19.186.35","virtualMachineId":"8635622412987052032","resourceGroupName":"默认资源组","regionOssName":"廊坊二区","natForwardRule":"empty","vaultBindStatus":"","provinceCode":"18","imageOs":"AnolisOS","userId":"679094","cloudRegionCode":"cn-langfang-2","createTime":"2025-07-21 21:03:59","regionId":"7961131233390559232","cloudName":"联通云","ramUnit":"G","vaultBindId":"","vaultBindType":""}],"pageNum":1}}
*/

func (region *SRegion) GetInstances(zoneId string, id string) ([]SInstance, error) {
	params := url.Values{}
	params.Set("cloudRegionCode", region.CloudRegionCode)
	if len(zoneId) > 0 {
		params.Set("zoneCode", zoneId)
	}
	if len(id) > 0 {
		params.Set("virtualMachineId", id)
	}
	resp, err := region.list("/instance/v1/product/ecs", params)
	if err != nil {
		return nil, err
	}
	ret := []SInstance{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) GetInstance(id string) (*SInstance, error) {
	vms, err := region.GetInstances("", id)
	if err != nil {
		return nil, err
	}
	for i := range vms {
		if vms[i].VirtualMachineId == id {
			return &vms[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetInstance")
}

func (ins *SInstance) AssignSecurityGroup(secgroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedVMChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) Refresh() error {
	vm, err := ins.host.zone.region.GetInstance(ins.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(ins, vm)
}

func (ins *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) GetBios() cloudprovider.TBiosType {
	return ""
}

func (ins *SInstance) GetBootOrder() string {
	return "bcd"
}

func (ins *SInstance) GetError() error {
	return nil
}

func (ins *SInstance) GetFullOsName() string {
	return ""
}

func (ins *SInstance) GetGlobalId() string {
	return ins.VirtualMachineId
}

func (ins *SInstance) GetId() string {
	return ins.VirtualMachineId
}

func (ins *SInstance) GetInstanceType() string {
	return ins.FlavorName
}

func (ins *SInstance) GetMachine() string {
	return "pc"
}

func (ins *SInstance) GetHostname() string {
	return ins.HostName
}

func (ins *SInstance) GetName() string {
	return ins.VirtualMachineName
}

func (ins *SInstance) GetOsArch() string {
	return ins.Architecture
}

func (ins *SInstance) GetOsDist() string {
	return ins.ImageOs
}

func (ins *SInstance) GetOsLang() string {
	return ""
}

func (ins *SInstance) GetOsType() cloudprovider.TOsType {
	if strings.Contains(strings.ToLower(ins.ImageOs), "windows") {
		return cloudprovider.OsTypeWindows
	}
	return cloudprovider.OsTypeLinux
}

func (ins *SInstance) GetOsVersion() string {
	return ins.ImageVersion
}

func (ins *SInstance) GetProjectId() string {
	return ""
}

func (ins *SInstance) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	for i := range ins.SecurityGroup {
		ret = append(ret, ins.SecurityGroup[i].SecurityGroupId)
	}
	return ret, nil
}

func (ins *SInstance) GetStatus() string {
	switch ins.Status {
	case "running":
		return api.VM_RUNNING
	case "stop":
		return api.VM_READY
	case "stop-ing":
		return api.VM_STOPPING
	case "start-ing":
		return api.VM_STARTING
	default:
		return strings.ToLower(ins.Status)
	}
}

func (ins *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_CUCLOUD
}

func (ins *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (ins *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (ins *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	ret := []cloudprovider.ICloudNic{}
	for i := range ins.NetCards {
		ret = append(ret, &ins.NetCards[i])
	}
	return ret, nil
}

func (ins *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (ins *SInstance) GetVcpuCount() int {
	return ins.CPU
}

func (ins *SInstance) GetVmemSizeMB() int {
	return ins.RAM * 1024
}

func (ins *SInstance) GetVdi() string {
	return ""
}

func (ins *SInstance) GetVga() string {
	return ""
}

func (ins *SInstance) RebuildRoot(ctx context.Context, opts *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (ins *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) StartVM(instanceId string) error {
	_, err := region.post(fmt.Sprintf("/instance/v1/product/ecs/%s/start", instanceId), map[string]interface{}{
		"cloudRegionCode": region.CloudRegionCode,
	})
	if err != nil {
		return err
	}
	return nil
}

func (ins *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	return ins.host.zone.region.StopVM(ins.InstanceId)
}

func (region *SRegion) StopVM(instanceId string) error {
	_, err := region.post(fmt.Sprintf("/instance/v1/product/ecs/%s/stop", instanceId), map[string]interface{}{
		"cloudRegionCode": region.CloudRegionCode,
	})
	if err != nil {
		return err
	}
	return nil
}

func (ins *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (ins *SInstance) GetIHost() cloudprovider.ICloudHost {
	return ins.host
}

func (ins *SInstance) DeployVM(ctx context.Context, opts *cloudprovider.SInstanceDeployOptions) error {
	return cloudprovider.ErrNotImplemented
}
