package azure

import (
	"fmt"
	"strings"
	"time"

	"context"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/secrules"
)

type ClassicVirtualMachineInstanceView struct {
	Status                   string   `json:"status,omitempty"`
	PowerState               string   `json:"powerState,omitempty"`
	PublicIpAddresses        []string `json:"publicIpAddresses,omitempty"`
	FullyQualifiedDomainName string   `json:"fullyQualifiedDomainName,omitempty"`

	UpdateDomain        int
	FaultDomain         int
	StatusMessage       string
	PrivateIpAddress    string           `json:"privateIpAddress,omitempty"`
	InstanceIpAddresses []string         `json:"instanceIpAddresses,omitempty"`
	ComputerName        string           `json:"computerName,omitempty"`
	GuestAgentStatus    GuestAgentStatus `json:"guestAgentStatus,omitempty"`
}

type SubResource struct {
	ID   string
	Name string
	Type string
}

type ClassicDisk struct {
	Lun             int32
	DiskName        string
	Caching         string
	OperatingSystem string
	IoType          string
	CreatedTime     string
	SourceImageName string
	VhdUri          string
	DiskSize        int32 `json:"diskSize,omitempty"`
	StorageAccount  SubResource
}

type ClassicStorageProfile struct {
	OperatingSystemDisk ClassicDisk    `json:"operatingSystemDisk,omitempty"`
	DataDisks           *[]ClassicDisk `json:"aataDisks,omitempty"`
}

type ClassicHardwareProfile struct {
	PlatformGuestAgent bool
	Size               string
	DeploymentName     string
	DeploymentId       string
	DeploymentLabel    string
	DeploymentLocked   bool
}

type InputEndpoint struct {
	EndpointName             string
	PrivatePort              int32
	PublicPort               int32
	Protocol                 string
	EnableDirectServerReturn bool
}

type InstanceIp struct {
	IdleTimeoutInMinutes int
	ID                   string
	Name                 string
	Type                 string
}

type ClassicVirtualNetwork struct {
	StaticIpAddress string   `json:"staticIpAddress,omitempty"`
	SubnetNames     []string `json:"subnetNames,omitempty"`
	ID              string
	Name            string
	Type            string
}

type ClassicNetworkProfile struct {
	InputEndpoints       *[]InputEndpoint      `json:"inputEndpoints,omitempty"`
	InstanceIps          *[]InstanceIp         `json:"instanceIps,omitempty"`
	ReservedIps          *[]SubResource        `json:"reservedIps,omitempty"`
	VirtualNetwork       ClassicVirtualNetwork `json:"virtualNetwork,omitempty"`
	NetworkSecurityGroup *SubResource          `json:"networkSecurityGroup,omitempty"`
}

type ClassicVirtualMachineProperties struct {
	InstanceView    *ClassicVirtualMachineInstanceView `json:"instanceView,omitempty"`
	NetworkProfile  ClassicNetworkProfile              `json:"networkProfile,omitempty"`
	HardwareProfile ClassicHardwareProfile             `json:"hardwareProfile,omitempty"`
	StorageProfile  ClassicStorageProfile              `json:"storageProfile,omitempty"`
}

type SClassicInstance struct {
	host *SClassicHost

	idisks []cloudprovider.ICloudDisk

	Properties ClassicVirtualMachineProperties `json:"properties,omitempty"`
	ID         string
	Name       string
	Type       string
	Location   string
}

func (self *SClassicInstance) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	priceKey := fmt.Sprintf("%s::%s", self.Properties.HardwareProfile.Size, self.host.zone.region.Name)
	data.Add(jsonutils.NewString(priceKey), "price_key")
	if self.Properties.NetworkProfile.NetworkSecurityGroup != nil {
		data.Add(jsonutils.NewString(self.Properties.NetworkProfile.NetworkSecurityGroup.ID), "secgroupId")
	}
	return data
}

func (self *SClassicInstance) GetHypervisor() string {
	return models.HYPERVISOR_AZURE
}

func (self *SClassicInstance) IsEmulated() bool {
	return false
}

func (self *SRegion) GetClassicInstances() ([]SClassicInstance, error) {
	result := []SClassicInstance{}
	instances := []SClassicInstance{}
	err := self.client.ListAll("Microsoft.ClassicCompute/virtualMachines", &instances)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(instances); i++ {
		if instances[i].Location == self.Name {
			result = append(result, instances[i])
		}
	}
	return result, nil
}

func (self *SRegion) GetClassicInstance(instanceId string) (*SClassicInstance, error) {
	instance := SClassicInstance{}
	return &instance, self.client.Get(instanceId, []string{"$expand=instanceView"}, &instance)
}

type ClassicInstanceDiskProperties struct {
	DiskName        string
	Caching         string
	OperatingSystem string
	IoType          string
	DiskSize        int32
	SourceImageName string
	VhdUri          string
}

type ClassicInstanceDisk struct {
	Properties ClassicInstanceDiskProperties
	ID         string
	Name       string
	Type       string
}

func (self *SClassicInstance) getDisks() ([]SClassicDisk, error) {
	disks := []SClassicDisk{}
	body, err := self.host.zone.region.client.jsonRequest("GET", fmt.Sprintf("%s/disks", self.ID), "")
	if err != nil {
		return nil, err
	}
	_disks, err := body.GetArray("value")
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(_disks); i++ {
		disk := SClassicDisk{}
		err = _disks[i].Unmarshal(&disk, "properties")
		if err != nil {
			return nil, err
		}
		storage := SClassicStorage{zone: self.host.zone, Name: disk.StorageAccount.Name, ID: disk.StorageAccount.ID}
		disk.DiskSizeGB = disk.DiskSize
		disk.storage = &storage
		disks = append(disks, disk)
	}
	return disks, nil
}

func (self *SClassicInstance) getNics() ([]SClassicInstanceNic, error) {
	instance, err := self.host.zone.region.GetClassicInstance(self.ID)
	if err != nil {
		return nil, err
	}
	networkProfile := instance.Properties.NetworkProfile
	ip, id := "", ""
	if len(networkProfile.VirtualNetwork.SubnetNames) > 0 {
		id = fmt.Sprintf("%s/%s", networkProfile.VirtualNetwork.ID, networkProfile.VirtualNetwork.SubnetNames[0])
	}
	if len(instance.Properties.NetworkProfile.VirtualNetwork.StaticIpAddress) > 0 {
		ip = instance.Properties.NetworkProfile.VirtualNetwork.StaticIpAddress
	}
	if (len(id) == 0 || len(ip) == 0) && instance.Properties.InstanceView != nil && len(instance.Properties.InstanceView.PrivateIpAddress) > 0 {
		if len(id) == 0 {
			id = fmt.Sprintf("%s/%s", self.ID, instance.Properties.InstanceView.PrivateIpAddress)
		}
		if len(ip) == 0 {
			ip = instance.Properties.InstanceView.PrivateIpAddress
		}
	}
	if len(id) > 0 && len(ip) > 0 {
		instanceNic := []SClassicInstanceNic{
			{instance: self, IP: ip, ID: id},
		}
		return instanceNic, nil
	}
	return nil, nil
}

func (self *SClassicInstance) Refresh() error {
	instance, err := self.host.zone.region.GetClassicInstance(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, instance)
}

func (self *SClassicInstance) GetStatus() string {
	if self.Properties.InstanceView == nil {
		err := self.Refresh()
		if err != nil {
			log.Errorf("failed to get status for classic instance %s", self.Name)
			return models.VM_UNKNOWN
		}
	}
	switch self.Properties.InstanceView.Status {
	case "StoppedDeallocated":
		return models.VM_READY
	case "ReadyRole":
		return models.VM_RUNNING
	case "Stopped":
		return models.VM_READY
	case "RoleStateUnknown":
		return models.VM_UNKNOWN
	default:
		log.Errorf("Unknow classic instance %s status %s", self.Name, self.Properties.InstanceView.Status)
		return models.VM_UNKNOWN
	}
}

func (self *SClassicInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SClassicInstance) AttachDisk(ctx context.Context, diskId string) error {
	if err := self.host.zone.region.AttachDisk(self.ID, diskId); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, self.GetStatus(), 10*time.Second, 300*time.Second)
}

func (self *SClassicInstance) DetachDisk(ctx context.Context, diskId string) error {
	if err := self.host.zone.region.DetachDisk(self.ID, diskId); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, self.GetStatus(), 10*time.Second, 300*time.Second)
}

func (self *SClassicInstance) ChangeConfig(ctx context.Context, ncpu int, vmem int) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicInstance) DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
	//return self.host.zone.region.DeployVM(self.ID, name, password, publicKey, deleteKeypair, description)
}

func (self *SClassicInstance) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
	//return self.host.zone.region.ReplaceSystemDisk(self.ID, imageId, passwd, publicKey, int32(sysSizeGB))
}

func (self *SClassicInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SClassicInstance) GetId() string {
	return self.ID
}

func (self *SClassicInstance) GetName() string {
	return self.Name
}

func (self *SClassicInstance) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SClassicInstance) DeleteVM(ctx context.Context) error {
	if err := self.host.zone.region.DeleteVM(self.ID); err != nil {
		return err
	}
	return nil
}

func (self *SClassicInstance) fetchDisks() error {
	disks, err := self.getDisks()
	if err != nil {
		return err
	}
	self.idisks = make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i++ {
		self.idisks[i] = &disks[i]
	}
	return nil
}

func (self *SClassicInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	if self.idisks == nil {
		if err := self.fetchDisks(); err != nil {
			return nil, err
		}
	}
	return self.idisks, nil
}

func (self *SClassicInstance) GetOSType() string {
	return osprofile.NormalizeOSType(self.Properties.StorageProfile.OperatingSystemDisk.OperatingSystem)
}

func (self *SClassicInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	instancenics := make([]cloudprovider.ICloudNic, 0)
	nics, err := self.getNics()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(nics); i++ {
		nics[i].instance = self
		instancenics = append(instancenics, &nics[i])
	}
	return instancenics, nil
}

func (self *SClassicInstance) GetOSName() string {
	return self.Properties.StorageProfile.OperatingSystemDisk.SourceImageName
}

func (self *SClassicInstance) GetBios() string {
	return "BIOS"
}

func (self *SClassicInstance) GetMachine() string {
	return "pc"
}

func (self *SClassicInstance) GetBootOrder() string {
	return "dcn"
}

func (self *SClassicInstance) GetVga() string {
	return "std"
}

func (self *SClassicInstance) GetVdi() string {
	return "vnc"
}

func (self *SClassicInstance) GetVcpuCount() int8 {
	if vmSize, ok := CLASSIC_VM_SIZES[self.Properties.HardwareProfile.Size]; ok {
		return vmSize.NumberOfCores
	}
	log.Errorf("failed to find classic VMSize for %s", self.Properties.HardwareProfile.Size)
	return 0
}

func (self *SClassicInstance) GetVmemSizeMB() int {
	if vmSize, ok := CLASSIC_VM_SIZES[self.Properties.HardwareProfile.Size]; ok {
		return vmSize.MemoryInMB
	}
	log.Errorf("failed to find classic VMSize for %s", self.Properties.HardwareProfile.Size)
	return 0
}

func (self *SClassicInstance) GetCreateTime() time.Time {
	return time.Now()
}

func (self *SClassicInstance) GetVNCInfo() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	return ret, nil
}

func (self *SClassicInstance) StartVM(ctx context.Context) error {
	if err := self.host.zone.region.StartVM(self.ID); err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.VM_RUNNING, 10*time.Second, 300*time.Second)
}

func (self *SClassicInstance) StopVM(ctx context.Context, isForce bool) error {
	err := self.host.zone.region.StopClassicVM(self.ID, isForce)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, models.VM_READY, 10*time.Second, 300*time.Second)
}

func (self *SRegion) StopClassicVM(instanceId string, isForce bool) error {
	_, err := self.client.PerformAction(instanceId, "shutdown", "")
	return err
}

func (self *SClassicInstance) SyncSecurityGroup(secgroupId string, name string, rules []secrules.SecurityRule) error {
	return cloudprovider.ErrNotSupported
}

func (self *SClassicInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if self.Properties.NetworkProfile.ReservedIps != nil && len(*self.Properties.NetworkProfile.ReservedIps) > 0 {
		for _, reserveIp := range *self.Properties.NetworkProfile.ReservedIps {
			eip, err := self.host.zone.region.GetClassicEip(reserveIp.ID)
			if err == nil {
				eip.instanceId = self.ID
				if eip.Properties.AttachedTo != nil && eip.Properties.AttachedTo.ID != self.ID {
					//一般是此实例deallocate, eip被绑到其他机器上了.
					return nil, nil
				}
				return eip, nil
			}
			log.Errorf("failed find eip %s for classic instance %s", reserveIp.Name, self.Name)
		}
	}
	if self.Properties.InstanceView != nil && len(self.Properties.InstanceView.PublicIpAddresses) > 0 {
		eip := SClassicEipAddress{
			region:     self.host.zone.region,
			ID:         self.ID,
			instanceId: self.ID,
			Name:       self.Properties.InstanceView.PublicIpAddresses[0],
			Properties: ClassicEipProperties{
				IpAddress: self.Properties.InstanceView.PublicIpAddresses[0],
			},
		}
		return &eip, nil
	}
	return nil, nil
}

func (self *SClassicInstance) GetBillingType() string {
	return models.BILLING_TYPE_POSTPAID
}

func (self *SClassicInstance) GetExpiredAt() time.Time {
	return time.Now()
}

func (self *SClassicInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SClassicInstance) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	return cloudprovider.ErrNotSupported
}
