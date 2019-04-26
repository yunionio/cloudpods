package zstack

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SInstanceCdrome struct {
}

type SInstance struct {
	host *SHost

	ZStackBasic
	ZoneUUID             string `json:"zoneUuid"`
	ClusterUUID          string `json:"clusterUuid"`
	HostUUID             string `json:"hostUuid"`
	LastHostUUID         string `json:"lastHostUuid"`
	RootVolumeUUID       string `json:"rootVolumeUuid"`
	Platform             string `json:"platform"`
	InstanceOfferingUUID string `json:"instanceOfferingUuid"`

	DefaultL3NetworkUUID string            `json:"defaultL3NetworkUuid"`
	Type                 string            `json:"type"`
	HypervisorType       string            `json:"hypervisorType"`
	MemorySize           int               `json:"memorySize"`
	CPUNum               int               `json:"cpuNum"`
	CPUSpeed             int               `json:"cpuSpeed"`
	State                string            `json:"state"`
	InternalID           string            `json:"internalId"`
	VMNics               []SInstanceNic    `json:"vmNics"`
	AllVolumes           []SDisk           `json:"allVolumes"`
	VMCdRoms             []SInstanceCdrome `json:"vmCdRoms"`
	ZStackTime
}

func (region *SRegion) GetInstance(instanceId string) (*SInstance, error) {
	instances, err := region.GetInstances("", instanceId, "")
	if err != nil {
		return nil, err
	}
	if len(instances) == 1 {
		if instances[0].UUID == instanceId {
			return &instances[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetInstances(hostId string, instanceId string, nicId string) ([]SInstance, error) {
	instance := []SInstance{}
	params := []string{"q=type=UserVm"}
	if len(hostId) > 0 {
		params = append(params, "q=lastHostUuid="+hostId)
	}
	if len(instanceId) > 0 {
		params = append(params, "q=uuid="+instanceId)
	}
	if len(nicId) > 0 {
		params = append(params, "q=vmNics.uuid="+nicId)
	}
	if SkipEsxi {
		params = append(params, "q=hypervisorType!=ESX")
	}
	return instance, region.client.listAll("vm-instances", params, &instance)
}

func (instance *SInstance) GetSecurityGroupIds() ([]string, error) {
	ids := []string{}
	secgroups, err := instance.host.zone.region.GetSecurityGroups("", instance.UUID)
	if err != nil {
		return nil, err
	}
	for _, secgroup := range secgroups {
		ids = append(ids, secgroup.UUID)
	}
	return ids, nil
}

func (instance *SInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()

	return data
}

func (instance *SInstance) GetCreateTime() time.Time {
	return instance.CreateDate
}

func (instance *SInstance) GetIHost() cloudprovider.ICloudHost {
	return instance.host
}

func (instance *SInstance) GetId() string {
	return instance.UUID
}

func (instance *SInstance) GetName() string {
	return instance.Name
}

func (instance *SInstance) GetGlobalId() string {
	return instance.GetId()
}

func (instance *SInstance) IsEmulated() bool {
	return false
}

func (instance *SInstance) GetInstanceType() string {
	if len(instance.InstanceOfferingUUID) > 0 {
		offer, err := instance.host.zone.region.GetInstanceOffering(instance.InstanceOfferingUUID)
		if err == nil {
			return offer.Name
		}
	}
	return instance.Type
}

func (instance *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	idisks := []cloudprovider.ICloudDisk{}
	rootDisk, err := instance.host.zone.region.GetDiskWithStorage(instance.RootVolumeUUID)
	if err != nil {
		return nil, err
	}
	idisks = append(idisks, rootDisk)
	for i := 0; i < len(instance.AllVolumes); i++ {
		if instance.AllVolumes[i].UUID != instance.RootVolumeUUID {
			dataDisk, err := instance.host.zone.region.GetDiskWithStorage(instance.AllVolumes[i].UUID)
			if err != nil {
				return nil, err
			}
			idisks = append(idisks, dataDisk)
		}
	}
	return idisks, nil
}

func (instance *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	iNics := []cloudprovider.ICloudNic{}
	for i := 0; i < len(instance.VMNics); i++ {
		instance.VMNics[i].instance = instance
		iNics = append(iNics, &instance.VMNics[i])
	}
	return iNics, nil
}

func (instance *SInstance) GetVcpuCount() int {
	return instance.CPUNum
}

func (instance *SInstance) GetVmemSizeMB() int {
	return instance.MemorySize / 1024 / 1024
}

func (instance *SInstance) GetBootOrder() string {
	return "dcn"
}

func (instance *SInstance) GetVga() string {
	return "std"
}

func (instance *SInstance) GetVdi() string {
	return "vnc"
}

func (instance *SInstance) GetOSType() string {
	return osprofile.NormalizeOSType(instance.Platform)
}

func (instance *SInstance) GetOSName() string {
	return instance.Platform
}

func (instance *SInstance) GetBios() string {
	return "BIOS"
}

func (instance *SInstance) GetMachine() string {
	return "pc"
}

func (instance *SInstance) GetStatus() string {
	switch instance.State {
	case "Stopped":
		return api.VM_READY
	case "Running":
		return api.VM_RUNNING
	case "Destroyed":
		return api.VM_DEALLOCATED
	default:
		log.Errorf("Unknown instance %s status %s", instance.Name, instance.State)
		return api.VM_UNKNOWN
	}
}

func (instance *SInstance) Refresh() error {
	new, err := instance.host.zone.region.GetInstance(instance.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(instance, new)
}

func (instance *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_ZSTACK
}

func (instance *SInstance) StartVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) StopVM(ctx context.Context, isForce bool) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	info, err := instance.host.zone.region.GetInstanceConsoleInfo(instance.UUID)
	if err != nil {
		return nil, err
	}
	authURL, _ := url.Parse(instance.host.zone.region.client.authURL)
	return jsonutils.Marshal(map[string]string{
		"url":         fmt.Sprintf("%s://%s:5000/thirdparty/vnc_auto.html?host=%s&port=%d&token=%s&title=%s", info.Scheme, authURL.Host, info.Hostname, info.Port, info.Token, instance.Name),
		"protocol":    "zstack",
		"instance_id": instance.UUID,
	}), nil
}

func (instance *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (instance *SInstance) ChangeConfig(ctx context.Context, ncpu int, vmem int) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) ChangeConfig2(ctx context.Context, instanceType string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) DeleteVM(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	eips, err := instance.host.zone.region.GetEips("", instance.UUID)
	if err != nil {
		return nil, err
	}
	if len(eips) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(eips) == 1 {
		return &eips[0], nil
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (instance *SInstance) AssignSecurityGroup(secgroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetBillingType() string {
	return ""
}

func (instance *SInstance) GetCreatedAt() time.Time {
	return instance.CreateDate
}

func (instance *SInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (instance *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstance) GetProjectId() string {
	return ""
}

func (instance *SInstance) GetError() error {
	return nil
}

type SConsoleInfo struct {
	Scheme   string `json:"scheme"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Token    string `json:"token"`
}

func (region *SRegion) GetInstanceConsoleInfo(instnaceId string) (*SConsoleInfo, error) {
	params := map[string]interface{}{
		"params": map[string]string{
			"vmInstanceUuid": instnaceId,
		},
	}
	resp, err := region.client.post("consoles", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	info := &SConsoleInfo{}
	err = resp.Unmarshal(info, "inventory")
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (region *SRegion) GetInstanceConsolePassword(instnaceId string) (string, error) {
	resp, err := region.client.get("vm-instances", instnaceId, "console-passwords")
	if err != nil {
		return "", err
	}
	if resp.Contains("consolePassword") {
		return resp.GetString("consolePassword")
	}
	return "", nil
}
