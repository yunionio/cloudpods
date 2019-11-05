// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SVMSize struct {
	//MaxDataDiskCount     int32 `json:"maxDataDiskCount,omitempty"` //Unmarshal会出错
	MemoryInMB           int32 `json:"memoryInMB,omitempty"`
	NumberOfCores        int   `json:"numberOfCores,omitempty"`
	Name                 string
	OsDiskSizeInMB       int32 `json:"osDiskSizeInMB,omitempty"`
	ResourceDiskSizeInMB int32 `json:"resourceDiskSizeInMB,omitempty"`
}

type SRegion struct {
	multicloud.SRegion
	client *SAzureClient

	izones       []cloudprovider.ICloudZone
	ivpcs        []cloudprovider.ICloudVpc
	iclassicVpcs []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	ID             string
	SubscriptionID string
	Name           string
	DisplayName    string
	Latitude       string
	Longitude      string
}

func (self *SRegion) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
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
			if info.MemoryInMB == int32(memMB) && info.NumberOfCores == cpu {
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
	return fmt.Sprintf("%s/%s", self.client.GetAccessEnv(), self.Name)
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_AZURE
}

func (self *SRegion) trimGeographicString(geographic string) string {
	return strings.TrimFunc(geographic, func(r rune) bool {
		return !((r >= '0' && r <= '9') || r == '.' || r == '-')
	})
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	info := cloudprovider.SGeographicInfo{}
	if geographicInfo, ok := AzureGeographicInfo[self.Name]; ok {
		info = geographicInfo
	}

	self.Latitude = self.trimGeographicString(self.Latitude)
	self.Longitude = self.trimGeographicString(self.Longitude)

	latitude, err := strconv.ParseFloat(self.Latitude, 32)
	if err != nil {
		log.Errorf("Parse azure region %s latitude %s error: %v", self.Name, self.Latitude, err)
	} else {
		info.Latitude = float32(latitude)
	}

	longitude, err := strconv.ParseFloat(self.Longitude, 32)
	if err != nil {
		log.Errorf("Parse azure region %s longitude %s error: %v", self.Name, self.Longitude, err)
	} else {
		info.Latitude = float32(longitude)
	}
	return info
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
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

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return self.GetInstance(id)
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
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

func (self *SRegion) CreateInstanceSimple(name string, imgId, osType string, cpu int, memMb int, sysDiskSizeGB int, storageType string, dataDiskSizesGB []int, networkId string, passwd string, publicKey string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		z := izones[i].(*SZone)
		log.Debugf("Search in zone %s", z.Name)
		net := z.getNetworkById(networkId)
		if net != nil {
			desc := &cloudprovider.SManagedVMCreateConfig{
				Name:              name,
				ExternalImageId:   imgId,
				SysDisk:           cloudprovider.SDiskInfo{SizeGB: sysDiskSizeGB, StorageType: storageType},
				Cpu:               cpu,
				MemoryMB:          memMb,
				ExternalNetworkId: networkId,
				Password:          seclib2.RandomPassword2(12),
				DataDisks:         []cloudprovider.SDiskInfo{},
				PublicKey:         publicKey,
				OsType:            osType,
			}
			if len(passwd) > 0 {
				desc.Password = passwd
			}
			for _, sizeGB := range dataDiskSizesGB {
				desc.DataDisks = append(desc.DataDisks, cloudprovider.SDiskInfo{SizeGB: sizeGB, StorageType: storageType})
			}
			host := z.getHost()
			inst, err := host.CreateVM(desc)
			if err != nil {
				return nil, err
			}
			instance := inst.(*SInstance)
			instance.host = host
			return instance, nil
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

func (region *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	if strings.Contains(strings.ToLower(secgroupId), "microsoft.classicnetwork") {
		return region.GetClassicSecurityGroupDetails(secgroupId)
	}
	return region.GetSecurityGroupDetails(secgroupId)
}

func (region *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	if conf.VpcId == "classic" {
		return region.CreateClassicSecurityGroup(conf.Desc)
	}
	return region.CreateSecurityGroup(conf.Name)
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

func (region *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	iBuckets, err := region.client.getIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "getIBuckets")
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range iBuckets {
		if iBuckets[i].GetLocation() != region.GetId() {
			continue
		}
		ret = append(ret, iBuckets[i])
	}
	return ret, nil
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, acl string) error {
	_, err := region.createStorageAccount(name, storageClassStr)
	if err != nil {
		return errors.Wrapf(err, "region.createStorageAccount name=%s storageClass=%s acl=%s", name, storageClassStr, acl)
	}
	return nil
}

func (region *SRegion) DeleteIBucket(name string) error {
	accounts, err := region.GetStorageAccounts()
	if err != nil {
		return errors.Wrap(err, "GetStorageAccounts")
	}
	for i := range accounts {
		if accounts[i].Name == name {
			err = region.client.Delete(accounts[i].ID)
			if err != nil {
				return errors.Wrap(err, "region.client.Delete")
			}
			return nil
		}
	}
	region.client.invalidateIBuckets()
	return nil
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	return region.checkStorageAccountNameExist(name)
}

func (region *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return cloudprovider.GetIBucketById(region, name)
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketById(name)
}
