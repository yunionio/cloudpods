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
	"net/url"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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

	storageCache *SStoragecache

	ID          string
	Name        string
	DisplayName string
	Latitude    string
	Longitude   string
}

// ///////////////////////////////////////////////////////////////////////////
func (self *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (self *SRegion) GetClient() *SAzureClient {
	return self.client
}

func (self *SRegion) ListVmSizes() ([]SVMSize, error) {
	result := []SVMSize{}
	resource := fmt.Sprintf("Microsoft.Compute/locations/%s/vmSizes", self.Name)
	return result, self.client.list(resource, url.Values{}, &result)
}

func (self *SRegion) getHardwareProfile(cpu, memMB int) []string {
	vmSizes, err := self.ListVmSizes()
	if err != nil {
		return []string{}
	}
	result := []string{}
	for i := range vmSizes {
		if vmSizes[i].MemoryInMB == int32(memMB) && vmSizes[i].NumberOfCores == cpu {
			result = append(result, vmSizes[i].Name)
		}
	}
	return result
}

func (self *SRegion) getVMSize(name string) (*SVMSize, error) {
	vmSizes, err := self.ListVmSizes()
	if err != nil {
		return nil, errors.Wrapf(err, "ListVmSizes")
	}
	for i := range vmSizes {
		if vmSizes[i].Name == name {
			return &vmSizes[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
}

func (self *SRegion) GetId() string {
	return self.Name
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_AZURE_CN, self.DisplayName)
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_AZURE_EN, self.DisplayName)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
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

func (self *SRegion) GetCloudEnv() string {
	return self.client.envName
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
		info.Longitude = float32(longitude)
	}
	return info
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	vpc := SVpc{
		region:   self,
		Name:     opts.NAME,
		Location: self.Name,
		Properties: VirtualNetworkPropertiesFormat{
			AddressSpace: AddressSpace{
				AddressPrefixes: []string{opts.CIDR},
			},
		},
		Type: "Microsoft.Network/virtualNetworks",
	}
	return &vpc, self.create("", jsonutils.Marshal(vpc), &vpc)
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

func (self *SRegion) getZone() *SZone {
	return &SZone{region: self, Name: self.Name}
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	return []cloudprovider.ICloudZone{self.getZone()}, nil
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (self *SRegion) ListVpcs() ([]SVpc, error) {
	result := []SVpc{}
	err := self.list("Microsoft.Network/virtualNetworks", url.Values{}, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return result, nil
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := self.ListVpcs()
	if err != nil {
		return nil, errors.Wrapf(err, "ListVpcs")
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		vpcs[i].region = self
		ret = append(ret, &vpcs[i])
	}
	return ret, nil
}

func (self *SRegion) CreateInstanceSimple(name string, imgId, osType string, cpu int, memMb int, sysDiskSizeGB int, storageType string, dataDiskSizesGB []int, nicId string, passwd string, publicKey string) (*SInstance, error) {
	desc := &cloudprovider.SManagedVMCreateConfig{
		Name:            name,
		ExternalImageId: imgId,
		SysDisk:         cloudprovider.SDiskInfo{SizeGB: sysDiskSizeGB, StorageType: storageType},
		Cpu:             cpu,
		MemoryMB:        memMb,
		Password:        passwd,
		DataDisks:       []cloudprovider.SDiskInfo{},
		PublicKey:       publicKey,
		OsType:          osType,
	}
	if len(passwd) > 0 {
		desc.Password = passwd
	}
	for _, sizeGB := range dataDiskSizesGB {
		desc.DataDisks = append(desc.DataDisks, cloudprovider.SDiskInfo{SizeGB: sizeGB, StorageType: storageType})
	}
	return self._createVM(desc, nicId)
}

func (region *SRegion) GetEips() ([]SEipAddress, error) {
	eips := []SEipAddress{}
	err := region.client.list("Microsoft.Network/publicIPAddresses", url.Values{}, &eips)
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
		return nil, errors.Wrapf(err, "GetEips")
	}
	ieips := []cloudprovider.ICloudEIP{}
	for i := 0; i < len(eips); i++ {
		if len(eips[i].GetIpAddr()) == 0 {
			continue
		}
		_, err := netutils.NewIPV4Addr(eips[i].GetIpAddr())
		if err != nil {
			continue
		}
		eips[i].region = region
		ieips = append(ieips, &eips[i])
	}
	return ieips, nil
}

func (self *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := self.ListSecgroups()
	if err != nil {
		return nil, errors.Wrapf(err, "ListSecgroups")
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range secgroups {
		secgroups[i].region = self
		ret = append(ret, &secgroups[i])
	}
	return ret, nil
}

func (region *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return region.GetSecurityGroupDetails(secgroupId)
}

func (region *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return region.CreateSecurityGroup(opts)
}

func (region *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	lbs, err := region.GetLoadbalancers()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudLoadbalancer{}
	for i := range lbs {
		lbs[i].region = region
		ret = append(ret, &lbs[i])
	}
	return ret, nil
}

func (region *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	lb := SLoadbalancer{}
	params := url.Values{}
	params.Set("api-version", "2021-02-01")
	err := region.get(loadbalancerId, params, &lb)
	if err != nil {
		return nil, errors.Wrapf(err, "get")
	}

	lb.region = region
	return &lb, nil
}

func (region *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotSupported, "GetILoadBalancerAclById")
}

func (region *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	segs := strings.Split(certId, "/sslCertificates")
	if len(segs[0]) > 0 {
		lb, err := region.GetLoadbalancer(segs[0])
		if err != nil {
			return nil, errors.Wrap(err, "GetILoadBalancerById")
		}
		for i := range lb.Properties.SSLCertificates {
			ssl := &lb.Properties.SSLCertificates[i]
			ssl.region = region
			if ssl.GetGlobalId() == certId {
				return ssl, nil
			}
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, certId)
}

func (region *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	lbs, err := region.GetLoadbalancerCertificates()
	if err != nil {
		return nil, errors.Wrap(err, "GetILoadBalancers")
	}

	ret := []cloudprovider.ICloudLoadbalancerCertificate{}
	for i := range lbs {
		lbs[i].region = region
		ret = append(ret, &lbs[i])
	}

	return ret, nil
}

func (region *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancerCertificate")
}

func (region *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotSupported, "GetILoadBalancerAcls")
}

func (region *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancerCreateOptions) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancer")
}

func (region *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancerAcl")
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	accounts, err := region.ListStorageAccounts()
	if err != nil {
		return nil, errors.Wrapf(err, "ListStorageAccounts")
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range accounts {
		ret = append(ret, &accounts[i])
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
	accounts, err := region.listStorageAccounts()
	if err != nil {
		return errors.Wrap(err, "ListStorageAccounts")
	}
	for i := range accounts {
		if accounts[i].Name == name {
			err = region.del(accounts[i].ID)
			if err != nil {
				return errors.Wrapf(err, "region.del")
			}
			return nil
		}
	}
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

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (self *SRegion) get(resource string, params url.Values, retVal interface{}) error {
	return self.client.get(resource, params, retVal)
}

func (self *SRegion) Show(resource string) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	err := self.get(resource, url.Values{}, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) del(resource string) error {
	return self.client.del(resource)
}

func (self *SRegion) Delete(resource string) error {
	return self.del(resource)
}

func (self *SRegion) checkResourceGroup(resourceGroup string) (string, error) {
	if len(resourceGroup) == 0 {
		resourceGroup = "Default"
	}
	resourceGroups, err := self.client.ListResourceGroups()
	if err != nil {
		return "", errors.Wrapf(err, "ListResourceGroups")
	}

	for i := range resourceGroups {
		proj := resourceGroups[i]
		if strings.ToLower(proj.GetName()) == strings.ToLower(resourceGroup) ||
			proj.GetGlobalId() == resourceGroup ||
			(strings.Contains(resourceGroup, "/") && strings.HasSuffix(proj.GetGlobalId(), resourceGroup)) {
			return resourceGroup, nil
		}
	}
	_, err = self.CreateResourceGroup(resourceGroup)
	return resourceGroup, err
}

type sInfo struct {
	Location string `json:"Location"`
	Name     string `json:"Name"`
	Type     string `json:"Type"`
}

func (self *SRegion) createInfo(body jsonutils.JSONObject) (sInfo, error) {
	info := sInfo{}
	err := body.Unmarshal(&info)
	if err != nil {
		return info, errors.Wrapf(err, "body.Unmarshal")
	}
	if len(info.Name) == 0 {
		return info, fmt.Errorf("Missing name params")
	}
	if len(info.Type) == 0 {
		return info, fmt.Errorf("Missing type params")
	}
	return info, nil
}

func (self *SRegion) create(resourceGroup string, _body jsonutils.JSONObject, retVal interface{}) error {
	body := _body.(*jsonutils.JSONDict)
	info, err := self.createInfo(_body)
	if err != nil {
		return errors.Wrapf(err, "createInfo")
	}
	resourceGroup, err = self.checkResourceGroup(resourceGroup)
	if err != nil {
		return errors.Wrapf(err, "checkResourceGroup(%s)", resourceGroup)
	}
	info.Name, err = self.client.getUniqName(resourceGroup, info.Type, info.Name)
	if err != nil {
		return errors.Wrapf(err, "getUniqName")
	}
	info.Location = self.Name
	body.Update(jsonutils.Marshal(info))
	body.Remove("location")
	body.Remove("name")
	body.Remove("type")
	return self.client.create(resourceGroup, info.Type, info.Name, body, retVal)
}

func (self *SRegion) update(body jsonutils.JSONObject, retVal interface{}) error {
	return self.client.update(body, retVal)
}

func (self *SRegion) perform(id, action string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.client.perform(id, action, body)
}

func (self *SRegion) put(resource string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.client.put(resource, body)
}

func (self *SRegion) patch(resource string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.client.patch(resource, body)
}

func (self *SRegion) list(resource string, params url.Values, retVal interface{}) error {
	result := []jsonutils.JSONObject{}
	err := self.client.list(resource, params, &result)
	if err != nil {
		return errors.Wrapf(err, "client.list")
	}
	ret := []jsonutils.JSONObject{}
	for i := range result {
		location, _ := result[i].GetString("location")
		if len(location) == 0 || strings.ToLower(self.Name) == strings.ToLower(strings.Replace(location, " ", "", -1)) {
			ret = append(ret, result[i])
		}
	}
	return jsonutils.Update(retVal, ret)
}

func (self *SRegion) list_resources(resource string, apiVersion string, params url.Values) (jsonutils.JSONObject, error) {
	if gotypes.IsNil(params) {
		params = url.Values{}
	}
	params.Add("$filter", fmt.Sprintf("location eq '%s'", self.Name))
	params.Add("$filter", fmt.Sprintf("resourceType eq '%s'", resource))
	return self.client.list_v2("resources", apiVersion, params)
}

func (self *SRegion) list_v2(resource string, apiVersion string, params url.Values) (jsonutils.JSONObject, error) {
	if gotypes.IsNil(params) {
		params = url.Values{}
	}
	return self.client.list_v2(resource, apiVersion, params)
}

func (self *SRegion) post_v2(resource string, apiVersion string, body map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.client.post_v2(resource, apiVersion, body)
}

func (self *SRegion) show(resource string, apiVersion string) (jsonutils.JSONObject, error) {
	return self.client.list_v2(resource, apiVersion, nil)
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudVM, 0)
	for i := range zones {
		hosts, err := zones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		for j := range hosts {
			ivms, err := hosts[j].GetIVMs()
			if err != nil {
				return nil, err
			}
			ret = append(ret, ivms...)
		}
	}
	return ret, nil
}
