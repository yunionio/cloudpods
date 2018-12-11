package azure

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/util/secrules"
)

type SVMSize struct {
	//MaxDataDiskCount     int32 `json:"maxDataDiskCount,omitempty"` //Unmarshal会出错
	MemoryInMB           int32 `json:"memoryInMB,omitempty"`
	NumberOfCores        int32 `json:"numberOfCores,omitempty"`
	Name                 string
	OsDiskSizeInMB       int32 `json:"osDiskSizeInMB,omitempty"`
	ResourceDiskSizeInMB int32 `json:"resourceDiskSizeInMB,omitempty"`
}

type SRegion struct {
	client *SAzureClient

	izones       []cloudprovider.ICloudZone
	ivpcs        []cloudprovider.ICloudVpc
	iclassicVpcs []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	ID             string
	SubscriptionID string
	Name           string
	DisplayName    string
	Latitude       float32
	Longitude      float32
}

/////////////////////////////////////////////////////////////////////////////
func (self *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (self *SRegion) GetClient() *SAzureClient {
	return self.client
}

func (self *SRegion) GetVMSize(location string) (map[string]SVMSize, error) {
	if len(location) == 0 {
		location = self.Name
	}
	body, err := self.client.ListVmSizes(location)
	if err != nil {
		return nil, err
	}
	vmSizes := []SVMSize{}
	err = body.Unmarshal(&vmSizes, "value")
	if err != nil {
		return nil, err
	}
	result := map[string]SVMSize{}
	for i := 0; i < len(vmSizes); i++ {
		result[vmSizes[i].Name] = vmSizes[i]
	}
	return result, nil
}

func (self *SRegion) getHardwareProfile(cpu, memMB int) []string {
	if vmSizes, err := self.GetVMSize(""); err != nil {
		return []string{}
	} else {
		profiles := make([]string, 0)
		for vmSize, info := range vmSizes {
			if info.MemoryInMB == int32(memMB) && info.NumberOfCores == int32(cpu) {
				profiles = append(profiles, vmSize)
			}
		}
		return profiles
	}
}

func (self *SRegion) getVMSize(size string) (*SVMSize, error) {
	vmSizes, err := self.GetVMSize("")
	if err != nil {
		return nil, err
	}
	vmSize, ok := vmSizes[size]
	if !ok {
		return nil, cloudprovider.ErrNotFound
	}
	return &vmSize, nil
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) GetId() string {
	return self.Name
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_AZURE_CN, self.DisplayName)
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_AZURE, self.Name)
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_AZURE
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	info := cloudprovider.SGeographicInfo{}
	info.Latitude = self.Latitude
	info.Longitude = self.Longitude
	return info
}

func (self *SRegion) GetStatus() string {
	return models.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	vpc := SVpc{
		region:   self,
		Name:     name,
		Location: self.Name,
		Properties: VirtualNetworkPropertiesFormat{
			AddressSpace: AddressSpace{
				AddressPrefixes: []string{cidr},
			},
		},
		Type: "Microsoft.Network/virtualNetworks",
	}
	return &vpc, self.client.Create(jsonutils.Marshal(vpc), &vpc)
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		ihost, err := izones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		istore, err := izones[i].GetIStorageById(id)
		if err == nil {
			return istore, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	iHosts := make([]cloudprovider.ICloudHost, 0)

	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneHost, err := izones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		iHosts = append(iHosts, iZoneHost...)
	}
	return iHosts, nil
}

func (self *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneStores, err := izones[i].GetIStorages()
		if err != nil {
			return nil, err
		}
		iStores = append(iStores, iZoneStores...)
	}
	return iStores, nil
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return self.storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(ivpcs); i++ {
		if ivpcs[i].GetGlobalId() == id {
			return ivpcs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	if izones, err := self.GetIZones(); err != nil {
		return nil, err
	} else {
		for i := 0; i < len(izones); i += 1 {
			if izones[i].GetGlobalId() == id {
				return izones[i], nil
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	if izones, err := self.GetIZones(); err != nil {
		return nil, err
	} else {
		for i := 0; i < len(izones); i += 1 {
			zone := izones[i].(*SZone)
			if zone.GetId() == id {
				return zone, nil
			}
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
}

func (self *SRegion) fetchZones() error {
	if self.izones == nil || len(self.izones) == 0 {
		self.izones = make([]cloudprovider.ICloudZone, 1)
		zone := SZone{region: self, Name: self.Name}
		self.izones[0] = &zone
	}
	return nil
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if self.izones == nil {
		if err := self.fetchInfrastructure(); err != nil {
			return nil, err
		}
	}
	return self.izones, nil
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (self *SRegion) getVpcs() ([]SVpc, error) {
	result := []SVpc{}
	vpcs := []SVpc{}
	err := self.client.ListAll("Microsoft.Network/virtualNetworks", &vpcs)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(vpcs); i++ {
		if vpcs[i].Location == self.Name {
			result = append(result, vpcs[i])
		}
	}
	return result, nil
}

func (self *SRegion) getClassicVpcs() ([]SClassicVpc, error) {
	result := []SClassicVpc{}
	for _, resourceType := range []string{"Microsoft.ClassicNetwork/virtualNetworks"} {
		vpcs := []SClassicVpc{}
		err := self.client.ListAll(resourceType, &vpcs)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(vpcs); i++ {
			if vpcs[i].Location == self.Name {
				result = append(result, vpcs[i])
			}
		}
	}
	return result, nil
}

func (self *SRegion) fetchIClassicVpc() error {
	classicVpcs, err := self.getClassicVpcs()
	if err != nil {
		return err
	}
	self.iclassicVpcs = make([]cloudprovider.ICloudVpc, 0)
	for i := 0; i < len(classicVpcs); i++ {
		classicVpcs[i].region = self
		self.iclassicVpcs = append(self.iclassicVpcs, &classicVpcs[i])
	}
	return nil
}

func (self *SRegion) fetchIVpc() error {
	vpcs, err := self.getVpcs()
	if err != nil {
		return err
	}
	self.ivpcs = make([]cloudprovider.ICloudVpc, 0)
	for i := 0; i < len(vpcs); i++ {
		if vpcs[i].Location == self.Name {
			vpcs[i].region = self
			self.ivpcs = append(self.ivpcs, &vpcs[i])
		}
	}
	return nil
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if self.ivpcs == nil || self.iclassicVpcs == nil {
		if err := self.fetchInfrastructure(); err != nil {
			return nil, err
		}
	}
	for _, vpc := range self.ivpcs {
		log.Debugf("find vpc %s for region %s", vpc.GetName(), self.GetName())
	}
	for _, vpc := range self.iclassicVpcs {
		log.Debugf("find classic vpc %s for region %s", vpc.GetName(), self.GetName())
	}
	ivpcs := self.ivpcs
	if len(self.iclassicVpcs) > 0 {
		ivpcs = append(ivpcs, self.iclassicVpcs...)
	}
	return ivpcs, nil
}

func (self *SRegion) fetchInfrastructure() error {
	err := self.fetchZones()
	if err != nil {
		return err
	}
	err = self.fetchIVpc()
	if err != nil {
		return err
	}
	for i := 0; i < len(self.ivpcs); i++ {
		for j := 0; j < len(self.izones); j++ {
			zone := self.izones[j].(*SZone)
			vpc := self.ivpcs[i].(*SVpc)
			wire := SWire{zone: zone, vpc: vpc}
			zone.addWire(&wire)
			vpc.addWire(&wire)
		}
	}

	err = self.fetchIClassicVpc()
	if err != nil {
		return err
	}
	for i := 0; i < len(self.iclassicVpcs); i++ {
		for j := 0; j < len(self.izones); j++ {
			zone := self.izones[j].(*SZone)
			vpc := self.iclassicVpcs[i].(*SClassicVpc)
			wire := SClassicWire{zone: zone, vpc: vpc}
			zone.addClassicWire(&wire)
			vpc.addWire(&wire)
		}
	}
	return nil
}

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, networkId string, passwd string, publicKey string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		z := izones[i].(*SZone)
		log.Debugf("Search in zone %s", z.Name)
		net := z.getNetworkById(networkId)
		if net != nil {
			passwd := seclib2.RandomPassword2(12)
			inst, err := z.getHost().CreateVM(name, imgId, 30, cpu, memGB*1024, networkId, "", "", passwd, storageType, dataDiskSizesGB, publicKey, "", "", nil)
			if err != nil {
				return nil, err
			}
			return inst.(*SInstance), nil
		}
	}
	return nil, fmt.Errorf("cannot find network %s", networkId)
}

func (region *SRegion) GetEips() ([]SEipAddress, error) {
	eips := []SEipAddress{}
	err := region.client.ListAll("Microsoft.Network/publicIPAddresses", &eips)
	if err != nil {
		return nil, err
	}
	result := []SEipAddress{}
	for i := 0; i < len(eips); i++ {
		if eips[i].Location == region.Name {
			eips[i].region = region
			result = append(result, eips[i])
		}
	}
	return result, nil
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := region.GetEips()
	if err != nil {
		return nil, err
	}
	classicEips, err := region.GetClassicEips()
	if err != nil {
		return nil, err
	}
	ieips := make([]cloudprovider.ICloudEIP, len(eips)+len(classicEips))
	for i := 0; i < len(eips); i++ {
		eips[i].region = region
		ieips[i] = &eips[i]
	}
	for i := 0; i < len(classicEips); i++ {
		classicEips[i].region = region
		ieips[len(eips)+i] = &classicEips[i]
	}
	return ieips, nil
}

func (region *SRegion) DeleteSecurityGroup(vpcId, secgroupId string) error {
	if vpcId == "classic" {
		return region.deleteClassicSecurityGroup(secgroupId)
	}
	secgroup, err := region.GetSecurityGroupDetails(secgroupId)
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			return nil
		}
		return err
	}
	if secgroup.Properties.NetworkInterfaces != nil {
		for _, nic := range *secgroup.Properties.NetworkInterfaces {
			nic, err := region.GetNetworkInterfaceDetail(nic.ID)
			if err != nil {
				return err
			}
			nic.Properties.NetworkSecurityGroup = nil
			if err := region.client.Update(jsonutils.Marshal(nic), nil); err != nil {
				return err
			}
		}
	}
	return region.client.Delete(secgroupId)
}

func (region *SRegion) SyncSecurityGroup(secgroupId, vpcId, name, desc string, rules []secrules.SecurityRule) (string, error) {
	if vpcId == "classic" {
		return region.syncClassicSecurityGroup(secgroupId, name, desc, rules)
	}
	if len(secgroupId) > 0 {
		if _, err := region.GetSecurityGroupDetails(secgroupId); err != nil {
			if err != cloudprovider.ErrNotFound {
				return "", err
			}
			secgroupId = ""
		}
	}
	if len(secgroupId) == 0 {
		secgroup, err := region.CreateSecurityGroup(name)
		if err != nil {
			return "", err
		}
		secgroupId = secgroup.ID
	}
	return region.updateSecurityGroupRules(secgroupId, rules)
}
