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

package qcloud

import (
	"context"
	"fmt"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion

	client *SQcloudClient

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	instanceTypes []SInstanceType

	Region      string
	RegionName  string
	RegionState string

	Latitude      float64
	Longitude     float64
	fetchLocation bool
}

func (self *SRegion) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetSkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	lbs, err := self.GetLoadbalancers(nil)
	if err != nil {
		return nil, err
	}

	ilbs := make([]cloudprovider.ICloudLoadbalancer, len(lbs))
	for i := range lbs {
		lbs[i].region = self
		ilbs[i] = &lbs[i]
	}

	return ilbs, nil
}

// 腾讯云不支持acl
func (self *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return []cloudprovider.ICloudLoadbalancerAcl{}, nil
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	lbs, err := self.GetLoadbalancers(nil)
	if err != nil {
		return nil, err
	}

	icerts := []cloudprovider.ICloudLoadbalancerCertificate{}
	for _, lb := range lbs {
		listeners, err := lb.GetLoadbalancerListeners("HTTPS")
		if err != nil {
			return nil, err
		}

		certIds := []string{}
		for _, listener := range listeners {
			if len(listener.Certificate.CERTID) > 0 && !utils.IsInStringArray(listener.Certificate.CERTID, certIds) {
				certIds = append(certIds, listener.Certificate.CERTID)
			}

			if len(listener.Certificate.CERTCAID) > 0 && !utils.IsInStringArray(listener.Certificate.CERTCAID, certIds) {
				certIds = append(certIds, listener.Certificate.CERTCAID)
			}

			for _, rule := range listener.Rules {
				if len(rule.Certificate.CERTID) > 0 && !utils.IsInStringArray(rule.Certificate.CERTID, certIds) {
					certIds = append(certIds, rule.Certificate.CERTID)
				}

				if len(rule.Certificate.CERTCAID) > 0 && !utils.IsInStringArray(rule.Certificate.CERTCAID, certIds) {
					certIds = append(certIds, rule.Certificate.CERTCAID)
				}
			}
		}

		for _, cid := range certIds {
			icert, err := self.GetILoadBalancerCertificateById(cid)
			if err != nil {
				return nil, err
			}

			icerts = append(icerts, icert)
		}
	}

	return icerts, nil
}

func (self *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	certs, _, err := self.GetCertificates(certId, true, 0, 0)
	if err != nil {
		return nil, err
	}

	icerts := []cloudprovider.ICloudLoadbalancerCertificate{}
	for i := 0; i < len(certs); i++ {
		cert := SLBCertificate{region: self, SCertificate: certs[i]}
		icerts = append(icerts, &cert)
	}

	if len(certs) == 1 {
		return icerts[0], nil
	} else if len(certs) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return nil, fmt.Errorf("GetILoadBalancerCertificateById %d certificate found, expect 1", len(certs))
	}
}

func (self *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	lbs, err := self.GetLoadbalancers([]string{loadbalancerId})
	if err != nil {
		return nil, err
	}

	if len(lbs) == 1 {
		lbs[0].region = self
		return &lbs[0], nil
	} else if len(lbs) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else {
		log.Debugf("GetILoadBalancerById %s %d loadbalancer found", loadbalancerId, len(lbs))
		return nil, cloudprovider.ErrNotFound
	}
}

func (self *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, nil
}

// https://cloud.tencent.com/document/api/214/30692
// todo: 1. 支持跨地域绑定负载均衡 及 https://cloud.tencent.com/document/product/214/12014
// todo: 2. 支持指定Project。 ProjectId可以通过 DescribeProject 接口获取。不填则属于默认项目。
func (self *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	LoadBalancerType := "INTERNAL"
	if loadbalancer.AddressType == api.LB_ADDR_TYPE_INTERNET {
		LoadBalancerType = "OPEN"
	}
	params := map[string]string{
		"LoadBalancerType": LoadBalancerType,
		"LoadBalancerName": loadbalancer.Name,
		"VpcId":            loadbalancer.VpcID,
	}

	if loadbalancer.AddressType != api.LB_ADDR_TYPE_INTERNET {
		params["SubnetId"] = loadbalancer.NetworkIDs[0]
	}

	resp, err := self.clbRequest("CreateLoadBalancer", params)
	if err != nil {
		return nil, err
	}

	requestId, err := resp.GetString("RequestId")
	if err != nil {
		return nil, err
	}

	lbs, err := resp.GetArray("LoadBalancerIds")
	if err != nil || len(lbs) != 1 {
		log.Debugf("CreateILoadBalancer %s", resp.String())
		return nil, err
	}

	err = self.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
	if err != nil {
		return nil, err
	}

	lbId, err := lbs[0].GetString()
	if err != nil {
		return nil, err
	}

	return self.GetLoadbalancer(lbId)
}

func (self *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

// todo:目前onecloud端只能指定服务器端证书。需要兼容客户端证书？
func (self *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	certId, err := self.CreateCertificate(cert.Certificate, "SVR", cert.PrivateKey, cert.Name)
	if err != nil {
		return nil, err
	}

	certs, _, err := self.GetCertificates(certId, false, 10, 0)
	if len(certs) != 1 || err != nil {
		log.Debugf("CreateILoadBalancerCertificate failed. %d certificate matched", len(certs))
		return nil, err
	}

	return &SLBCertificate{region: self, SCertificate: certs[0]}, nil
}

func (self *SRegion) GetId() string {
	return self.Region
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_QCLOUD_CN, self.RegionName)
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_QCLOUD, self.Region)
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_QCLOUD
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	params := make(map[string]string)
	if len(cidr) > 0 {
		params["CidrBlock"] = cidr
	}
	if len(name) > 0 {
		params["VpcName"] = name
	}
	body, err := self.vpcRequest("CreateVpc", params)
	if err != nil {
		return nil, err
	}
	vpcId, err := body.GetString("Vpc", "VpcId")
	if err != nil {
		return nil, err
	}
	err = self.fetchInfrastructure()
	if err != nil {
		return nil, err
	}
	return self.GetIVpcById(vpcId)
}

func (self *SRegion) GetCosClient(bucket *SBucket) (*cos.Client, error) {
	return self.client.getCosClient(bucket)
}

func (self *SRegion) GetClient() *SQcloudClient {
	return self.client
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return self.GetInstance(id)
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
}

func (self *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	if len(eipId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	eips, total, err := self.GetEips(eipId, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return &eips[0], nil
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, total, err := self.GetEips("", "", 0, 50)
	if err != nil {
		return nil, err
	}
	for len(eips) < total {
		var parts []SEipAddress
		parts, total, err = self.GetEips("", "", len(eips), 50)
		if err != nil {
			return nil, err
		}
		eips = append(eips, parts...)
	}
	ret := make([]cloudprovider.ICloudEIP, len(eips))
	for i := 0; i < len(eips); i++ {
		ret[i] = &eips[i]
	}
	return ret, nil
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

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}

func (self *SRegion) updateInstance(instId string, name, desc, passwd, hostname string) error {
	params := make(map[string]string)
	params["InstanceId"] = instId
	if len(name) > 0 {
		params["InstanceName"] = name
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	if len(passwd) > 0 {
		params["Password"] = passwd
	}
	if len(hostname) > 0 {
		params["HostName"] = hostname
	}
	_, err := self.cvmRequest("ModifyInstanceAttribute", params, true)
	return err
}

func (self *SRegion) UpdateInstancePassword(instId string, passwd string) error {
	return self.updateInstance(instId, "", "", passwd, "")
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

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		zone := izones[i].(*SZone)
		if zone.Zone == id {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if self.ivpcs == nil {
		err := self.fetchInfrastructure()
		if err != nil {
			return nil, err
		}
	}
	return self.ivpcs, nil
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

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if self.izones == nil {
		var err error
		err = self.fetchInfrastructure()
		if err != nil {
			return nil, err
		}
	}
	return self.izones, nil
}

func (self *SRegion) _fetchZones() error {
	params := make(map[string]string)
	zones := make([]SZone, 0)
	body, err := self.cvmRequest("DescribeZones", params, true)
	if err != nil {
		return err
	}
	err = body.Unmarshal(&zones, "ZoneSet")
	if err != nil {
		return err
	}
	self.izones = make([]cloudprovider.ICloudZone, len(zones))
	for i := 0; i < len(zones); i++ {
		zones[i].region = self
		self.izones[i] = &zones[i]
	}
	return nil
}

func (self *SRegion) fetchInfrastructure() error {
	err := self._fetchZones()
	if err != nil {
		return err
	}
	err = self.fetchIVpcs()
	if err != nil {
		return err
	}
	for i := 0; i < len(self.ivpcs); i += 1 {
		for j := 0; j < len(self.izones); j += 1 {
			zone := self.izones[j].(*SZone)
			vpc := self.ivpcs[i].(*SVpc)
			wire := SWire{zone: zone, vpc: vpc}
			zone.addWire(&wire)
			vpc.addWire(&wire)
		}
	}
	return nil
}

func (self *SRegion) DeleteVpc(vpcId string) error {
	params := make(map[string]string)
	params["VpcId"] = vpcId

	_, err := self.vpcRequest("DeleteVpc", params)
	return err
}

func (self *SRegion) getVpc(vpcId string) (*SVpc, error) {
	vpcs, total, err := self.GetVpcs([]string{vpcId}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	vpcs[0].region = self
	return &vpcs[0], nil
}

func (self *SRegion) fetchIVpcs() error {
	vpcs := make([]SVpc, 0)
	for {
		part, total, err := self.GetVpcs(nil, len(vpcs), 50)
		if err != nil {
			return err
		}
		vpcs = append(vpcs, part...)
		if len(vpcs) >= total {
			break
		}
	}
	self.ivpcs = make([]cloudprovider.ICloudVpc, len(vpcs))
	for i := 0; i < len(vpcs); i += 1 {
		vpcs[i].region = self
		self.ivpcs[i] = &vpcs[i]
	}
	return nil
}

func (self *SRegion) GetVpcs(vpcIds []string, offset int, limit int) ([]SVpc, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	if vpcIds != nil && len(vpcIds) > 0 {
		for index, vpcId := range vpcIds {
			params[fmt.Sprintf("VpcIds.%d", index)] = vpcId
		}
	}
	body, err := self.vpcRequest("DescribeVpcs", params)
	if err != nil {
		return nil, 0, err
	}
	vpcs := make([]SVpc, 0)
	err = body.Unmarshal(&vpcs, "VpcSet")
	if err != nil {
		log.Errorf("Unmarshal vpc fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return vpcs, int(total), nil
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.Region]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) GetStatus() string {
	if self.RegionState == "AVAILABLE" {
		return api.CLOUD_REGION_STATUS_INSERVER
	}
	return api.CLOUD_REGION_STATUS_OUTOFSERVICE
}

func (self *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (self *SRegion) vpcRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.vpcRequest(apiName, params)
}

func (self *SRegion) auditRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.auditRequest(apiName, params)
}

func (self *SRegion) vpc2017Request(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.vpc2017Request(apiName, params)
}

func (self *SRegion) cvmRequest(apiName string, params map[string]string, retry bool) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.jsonRequest(apiName, params, retry)
}

func (self *SRegion) accountRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return self.client.accountRequestRequest(apiName, params)
}

func (self *SRegion) cbsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.cbsRequest(apiName, params)
}

func (self *SRegion) clbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.clbRequest(apiName, params)
}

func (self *SRegion) lbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	params["Region"] = self.Region
	return self.client.lbRequest(apiName, params)
}

func (self *SRegion) wssRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return self.client.wssRequest(apiName, params)
}

func (self *SRegion) GetNetworks(ids []string, vpcId string, offset int, limit int) ([]SNetwork, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	base := 0
	if ids != nil && len(ids) > 0 {
		for index, networkId := range ids {
			params[fmt.Sprintf("SubnetIds.%d", index)] = networkId
		}
		base += len(ids)
	}
	if len(vpcId) > 0 {
		params["Filters.0.Name"] = "vpc-id"
		params["Filters.0.Values.0"] = vpcId
	}

	body, err := self.vpcRequest("DescribeSubnets", params)
	if err != nil {
		log.Errorf("DescribeSubnets fail %s", err)
		return nil, 0, err
	}

	networks := make([]SNetwork, 0)
	err = body.Unmarshal(&networks, "SubnetSet")
	if err != nil {
		log.Errorf("Unmarshal network fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return networks, int(total), nil
}

func (self *SRegion) GetNetwork(networkId string) (*SNetwork, error) {
	networks, total, err := self.GetNetworks([]string{networkId}, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if total > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	if total == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return &networks[0], nil
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (self *SRegion) GetMatchInstanceTypes(cpu int, memMB int, gpu int, zoneId string) ([]SInstanceType, error) {
	if self.instanceTypes == nil {
		types, err := self.GetInstanceTypes()
		if err != nil {
			log.Errorf("GetInstanceTypes %s", err)
			return nil, err
		}
		self.instanceTypes = types
	}

	var available []string
	if len(zoneId) > 0 {
		zone, err := self.getZoneById(zoneId)
		if err != nil {
			return nil, err
		}
		available = zone.getAvaliableInstanceTypes()
	}
	ret := make([]SInstanceType, 0)
	for _, t := range self.instanceTypes {
		if t.CPU == cpu && memMB == t.memoryMB() && gpu == t.GPU {
			if available == nil || utils.IsInStringArray(t.InstanceType, available) {
				ret = append(ret, t)
			}
		}
	}
	return ret, nil
}

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, networkId string, passwd string, publicKey string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		z := izones[i].(*SZone)
		log.Debugf("Search in zone %s", z.Zone)
		net := z.getNetworkById(networkId)
		if net != nil {
			desc := &cloudprovider.SManagedVMCreateConfig{
				Name:              name,
				ExternalImageId:   imgId,
				SysDisk:           cloudprovider.SDiskInfo{SizeGB: 0, StorageType: storageType},
				Cpu:               cpu,
				MemoryMB:          memGB * 1024,
				ExternalNetworkId: networkId,
				Password:          passwd,
				DataDisks:         []cloudprovider.SDiskInfo{},
				PublicKey:         publicKey,
			}
			for _, sizeGB := range dataDiskSizesGB {
				desc.DataDisks = append(desc.DataDisks, cloudprovider.SDiskInfo{SizeGB: sizeGB, StorageType: storageType})
			}
			inst, err := z.getHost().CreateVM(desc)
			if err != nil {
				return nil, err
			}
			return inst.(*SInstance), nil
		}
	}
	return nil, fmt.Errorf("cannot find network %s", networkId)
}

func (self *SRegion) instanceOperation(instanceId string, opname string, extra map[string]string, retry bool) error {
	params := make(map[string]string)
	params["InstanceIds.0"] = instanceId
	if extra != nil && len(extra) > 0 {
		for k, v := range extra {
			params[k] = v
		}
	}
	_, err := self.cvmRequest(opname, params, retry)
	return err
}

func (self *SRegion) GetInstanceVNCUrl(instanceId string) (string, error) {
	params := make(map[string]string)
	params["InstanceId"] = instanceId
	body, err := self.cvmRequest("DescribeInstanceVncUrl", params, true)
	if err != nil {
		return "", err
	}
	return body.GetString("InstanceVncUrl")
}

func (self *SRegion) GetInstanceStatus(instanceId string) (string, error) {
	instance, err := self.GetInstance(instanceId)
	if err != nil {
		return "", err
	}
	return instance.InstanceState, nil
}

func (self *SRegion) QueryAccountBalance() (*SAccountBalance, error) {
	return self.client.QueryAccountBalance()
}

func (self *SRegion) getCosEndpoint() string {
	return fmt.Sprintf("cos.%s.myqcloud.com", self.GetId())
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	iBuckets, err := region.client.getIBuckets()
	if err != nil {
		return nil, err
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

func (region *SRegion) CreateIBucket(name string, storageClassStr string, aclStr string) error {
	bucket := &SBucket{
		region: region,
		Name:   name,
	}
	coscli, err := region.GetCosClient(bucket)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	opts := &cos.BucketPutOptions{}
	if len(aclStr) > 0 {
		if utils.IsInStringArray(aclStr, []string{
			"private", "public-read", "public-read-write", "authenticated-read",
		}) {
			opts.XCosACL = aclStr
		} else {
			return errors.Error("invalid acl")
		}
	}
	_, err = coscli.Bucket.Put(context.Background(), opts)
	if err != nil {
		return errors.Wrap(err, "coscli.Bucket.Put")
	}
	region.client.invalidateIBuckets()
	return nil
}

func cosHttpCode(err error) int {
	if httpErr, ok := err.(*cos.ErrorResponse); ok {
		return httpErr.Response.StatusCode
	}
	return -1
}

func (region *SRegion) DeleteIBucket(name string) error {
	bucket := &SBucket{
		region: region,
		Name:   name,
	}
	coscli, err := region.GetCosClient(bucket)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	_, err = coscli.Bucket.Delete(context.Background())
	if err != nil {
		if cosHttpCode(err) == 404 {
			return nil
		}
		return errors.Wrap(err, "DeleteBucket")
	}
	return nil
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	bucket := &SBucket{
		region: region,
		Name:   name,
	}
	coscli, err := region.GetCosClient(bucket)
	if err != nil {
		return false, errors.Wrap(err, "GetCosClient")
	}
	_, err = coscli.Bucket.Head(context.Background())
	if err != nil {
		if cosHttpCode(err) == 404 {
			return false, nil
		}
		return false, errors.Wrap(err, "BucketExists")
	}
	return true, nil
}

func (region *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return cloudprovider.GetIBucketById(region, name)
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketById(name)
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return self.GetSecurityGroupDetails(secgroupId)
}

func (self *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return self.CreateSecurityGroup(conf.Name, conf.Desc)
}
