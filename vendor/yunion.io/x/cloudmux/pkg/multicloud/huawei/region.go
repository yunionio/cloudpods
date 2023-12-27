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

package huawei

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/obs"
)

type Locales struct {
	EnUs string `json:"en-us"`
	ZhCN string `json:"zh-cn"`
}

type SRegion struct {
	multicloud.SRegion

	client    *SHuaweiClient
	obsClient *obs.ObsClient // 对象存储client.请勿直接引用。

	Description    string
	Id             string
	Locales        Locales
	ParentRegionId string
	Type           string

	storageCache *SStoragecache
}

func (self *SRegion) GetClient() *SHuaweiClient {
	return self.client
}

func (self *SRegion) list(service, resource string, query url.Values) (jsonutils.JSONObject, error) {
	return self.client.list(service, self.Id, resource, query)
}

func (self *SRegion) delete(service, resource string) (jsonutils.JSONObject, error) {
	return self.client.delete(service, self.Id, resource)
}

func (self *SRegion) put(service, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.client.put(service, self.Id, resource, params)
}

func (self *SRegion) post(service, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.client.post(service, self.Id, resource, params)
}

func (self *SRegion) patch(service, resource string, query url.Values, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.client.patch(service, self.Id, resource, query, params)
}

func (self *SRegion) getOBSEndpoint() string {
	return getOBSEndpoint(self.getId())
}

func (self *SRegion) getOBSClient(signType obs.SignatureType) (*obs.ObsClient, error) {
	if self.obsClient == nil {
		obsClient, err := self.client.getOBSClient(self.getId(), signType)
		if err != nil {
			return nil, err
		}

		self.obsClient = obsClient
	}

	return self.obsClient, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=NovaListAvailabilityZones
func (self *SRegion) GetZones() ([]SZone, error) {
	resp, err := self.list(SERVICE_ECS_V2_1, "os-availability-zone", nil)
	if err != nil {
		return nil, err
	}
	ret := []SZone{}
	err = resp.Unmarshal(&ret, "availabilityZoneInfo")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	instance, err := self.GetInstance(id)
	if err != nil {
		return nil, err
	}
	return instance, err
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.getId()]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	elbs, err := self.GetLoadBalancers()
	if err != nil {
		return nil, err
	}

	ielbs := make([]cloudprovider.ICloudLoadbalancer, len(elbs))
	for i := range elbs {
		elbs[i].region = self
		ielbs[i] = &elbs[i]
	}

	return ielbs, nil
}

func (self *SRegion) GetLoadBalancers() ([]SLoadbalancer, error) {
	lbs := []SLoadbalancer{}
	params := url.Values{}
	return lbs, self.lbListAll("elb/loadbalancers", params, "loadbalancers", &lbs)
}

func (self *SRegion) GetILoadBalancerById(id string) (cloudprovider.ICloudLoadbalancer, error) {
	elb, err := self.GetLoadbalancer(id)
	if err != nil {
		return nil, err
	}
	return elb, nil
}

func (self *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	acl, err := self.GetLoadBalancerAcl(aclId)
	if err != nil {
		return nil, err
	}
	return acl, nil
}

func (self *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	cert, err := self.GetLoadBalancerCertificate(certId)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func (self *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	ret, err := self.CreateLoadBalancerCertificate(cert)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	ret, err := self.GetLoadBalancerAcls("")
	if err != nil {
		return nil, err
	}

	iret := make([]cloudprovider.ICloudLoadbalancerAcl, len(ret))
	for i := range ret {
		ret[i].region = self
		iret[i] = &ret[i]
	}
	return iret, nil
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	ret, err := self.GetLoadBalancerCertificates()
	if err != nil {
		return nil, err
	}

	iret := make([]cloudprovider.ICloudLoadbalancerCertificate, len(ret))
	for i := range ret {
		ret[i].region = self
		iret[i] = &ret[i]
	}
	return iret, nil
}

func (self *SRegion) GetId() string {
	return self.Id
}

func (self *SRegion) GetName() string {
	name := self.Locales.ZhCN
	suffix := self.getSuffix()
	if len(suffix) > 0 {
		name = fmt.Sprintf("%s-%s", name, suffix)
	}
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_HUAWEI_CN, name)
}

func (self *SRegion) getId() string {
	idx := strings.Index(self.Id, "_")
	if idx > 0 {
		return self.Id[:idx]
	}
	return self.Id
}

func (self *SRegion) getSuffix() string {
	idx := strings.Index(self.Id, "_")
	if idx > 0 {
		return self.Id[idx+1:]
	}
	return ""
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	en := self.Locales.EnUs
	suffix := self.getSuffix()
	if len(suffix) > 0 {
		en = fmt.Sprintf("%s-%s", en, suffix)
	}
	en = fmt.Sprintf("%s %s", CLOUD_PROVIDER_HUAWEI_EN, en)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", api.CLOUD_PROVIDER_HUAWEI, self.Id)
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) Refresh() error {
	return nil
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := self.GetZones()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		zones[i].region = self
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpcs")
	}
	ret := []cloudprovider.ICloudVpc{}
	for i := range vpcs {
		vpcs[i].region = self
		ret = append(ret, &vpcs[i])
	}
	return ret, nil
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := self.GetEips("", nil)
	if err != nil {
		return nil, err
	}

	ret := []cloudprovider.ICloudEIP{}
	for i := 0; i < len(eips); i += 1 {
		eips[i].region = self
		ret = append(ret, &eips[i])
	}
	return ret, nil
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(ivpcs); i += 1 {
		if ivpcs[i].GetGlobalId() == id {
			return ivpcs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	eip, err := self.GetEip(eipId)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (self *SRegion) DeleteSecurityGroup(id string) error {
	_, err := self.delete(SERVICE_VPC_V3, "vpc/security-groups/"+id)
	return err
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return self.GetSecurityGroup(secgroupId)
}

func (self *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	groups, err := self.GetSecurityGroups("")
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurityGroups")
	}
	ret := []cloudprovider.ICloudSecurityGroup{}
	for i := range groups {
		groups[i].region = self
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (self *SRegion) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return self.CreateSecurityGroup(opts)
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return self.CreateVpc(opts.NAME, opts.CIDR, opts.Desc)
}

func (self *SRegion) CreateVpc(name, cidr, desc string) (*SVpc, error) {
	params := map[string]interface{}{
		"vpc": map[string]string{
			"name":        name,
			"cidr":        cidr,
			"description": desc,
		},
	}
	vpc := &SVpc{region: self}
	resp, err := self.post(SERVICE_VPC, "vpcs", params)
	if err != nil {
		return nil, errors.Wrapf(err, "create vpc")
	}
	err = resp.Unmarshal(vpc, "vpc")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return vpc, nil
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090596.html
// size: 1Mbit/s~2000Mbit/s
// bgpType: 5_telcom，5_union，5_bgp，5_sbgp.
// 东北-大连：5_telcom、5_union
// 华南-广州：5_sbgp
// 华东-上海二：5_sbgp
// 华北-北京一：5_bgp、5_sbgp
// 亚太-香港：5_bgp
func (self *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	eip, err := self.AllocateEIP(opts)
	if err != nil {
		return nil, err
	}
	err = cloudprovider.WaitStatus(eip, api.EIP_STATUS_READY, 5*time.Second, time.Minute)
	return eip, err
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapshots("", "")
	if err != nil {
		log.Errorf("self.GetSnapshots fail %s", err)
		return nil, err
	}

	ret := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i += 1 {
		snapshots[i].region = self
		ret[i] = &snapshots[i]
	}
	return ret, nil
}

func (self *SRegion) GetISnapshotById(id string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
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

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		ihost, err := izones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
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

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		istore, err := izones[i].GetIStorageById(id)
		if err == nil {
			return istore, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_HUAWEI
}

func (self *SRegion) GetCloudEnv() string {
	return CLOUD_PROVIDER_HUAWEI
}

func (self *SRegion) CreateSecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (*SSecurityGroup, error) {
	params := map[string]interface{}{
		"name":                  opts.Name,
		"description":           opts.Desc,
		"enterprise_project_id": "0",
	}
	if len(opts.ProjectId) > 0 {
		params["enterprise_project_id"] = opts.ProjectId
	}
	resp, err := self.post(SERVICE_VPC_V3, "vpc/security-groups", map[string]interface{}{"security_group": params})
	if err != nil {
		return nil, err
	}
	ret := &SSecurityGroup{region: self}
	return ret, resp.Unmarshal(ret, "security_group")
}

func (self *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancerCreateOptions) (cloudprovider.ICloudLoadbalancer, error) {
	ret, err := self.CreateLoadBalancer(loadbalancer)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (self *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	ret, err := self.CreateLoadBalancerAcl(acl)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	iBuckets, err := region.client.getIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "getIBuckets")
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range iBuckets {
		// huawei OBS is shared across projects
		if iBuckets[i].GetLocation() == region.GetId() {
			ret = append(ret, iBuckets[i])
		}
	}
	return ret, nil
}

func str2StorageClass(storageClassStr string) (obs.StorageClassType, error) {
	if strings.EqualFold(storageClassStr, string(obs.StorageClassStandard)) {
		return obs.StorageClassStandard, nil
	} else if strings.EqualFold(storageClassStr, string(obs.StorageClassWarm)) {
		return obs.StorageClassWarm, nil
	} else if strings.EqualFold(storageClassStr, string(obs.StorageClassCold)) {
		return obs.StorageClassCold, nil
	} else {
		return obs.StorageClassStandard, errors.Error("unsupported storageClass")
	}
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, aclStr string) error {
	obsClient, err := region.getOBSClient("")
	if err != nil {
		return errors.Wrap(err, "region.getOBSClient")
	}
	input := &obs.CreateBucketInput{}
	input.Bucket = name
	input.Location = region.getId()
	if len(aclStr) > 0 {
		if strings.EqualFold(aclStr, string(obs.AclPrivate)) {
			input.ACL = obs.AclPrivate
		} else if strings.EqualFold(aclStr, string(obs.AclPublicRead)) {
			input.ACL = obs.AclPublicRead
		} else if strings.EqualFold(aclStr, string(obs.AclPublicReadWrite)) {
			input.ACL = obs.AclPublicReadWrite
		} else {
			return errors.Error("unsupported acl")
		}
	}
	if len(storageClassStr) > 0 {
		input.StorageClass, err = str2StorageClass(storageClassStr)
		if err != nil {
			return err
		}
	}
	_, err = obsClient.CreateBucket(input)
	if err != nil {
		return errors.Wrap(err, "obsClient.CreateBucket")
	}
	region.client.invalidateIBuckets()
	return nil
}

func obsHttpCode(err error) int {
	switch httpErr := err.(type) {
	case obs.ObsError:
		return httpErr.StatusCode
	case *obs.ObsError:
		return httpErr.StatusCode
	}
	return -1
}

func (region *SRegion) DeleteIBucket(name string) error {
	obsClient, err := region.getOBSClient("")
	if err != nil {
		return errors.Wrap(err, "region.getOBSClient")
	}
	_, err = obsClient.DeleteBucket(name)
	if err != nil {
		if obsHttpCode(err) == 404 {
			return nil
		}
		log.Debugf("%#v %s", err, err)
		return errors.Wrap(err, "DeleteBucket")
	}
	region.client.invalidateIBuckets()
	return nil
}

func (region *SRegion) HeadBucket(name string) (*obs.BaseModel, error) {
	obsClient, err := region.getOBSClient("")
	if err != nil {
		return nil, errors.Wrap(err, "region.getOBSClient")
	}
	return obsClient.HeadBucket(name)
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	_, err := region.HeadBucket(name)
	if err != nil {
		if obsHttpCode(err) == 404 {
			return false, nil
		} else {
			return false, errors.Wrap(err, "HeadBucket")
		}
	}
	return true, nil
}

func (region *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return cloudprovider.GetIBucketById(region, name)
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketById(name)
}

func (self *SRegion) GetSkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	caches, err := self.GetElasticCaches()
	if err != nil {
		return nil, err
	}

	icaches := make([]cloudprovider.ICloudElasticcache, len(caches))
	for i := range caches {
		caches[i].region = self
		icaches[i] = &caches[i]
	}

	return icaches, nil
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (self *SRegion) GetDiskTypes() ([]SDiskType, error) {
	resp, err := self.list(SERVICE_EVS, "types", nil)
	if err != nil {
		return nil, err
	}
	ret := []SDiskType{}
	err = resp.Unmarshal(&ret, "volume_types")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := region.GetInstances("")
	if err != nil {
		return nil, errors.Wrap(err, "GetInstances")
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		ret = append(ret, &vms[i])
	}
	return ret, nil
}
