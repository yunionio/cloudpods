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

package openstack

import (
	"fmt"
	"io"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion

	client *SOpenStackClient

	Name string

	vpcs []SVpc

	storageCache *SStoragecache
	routers      []SRouter
}

func (region *SRegion) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	backendGroups := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	pools, err := region.GetLoadbalancerPools()
	if err != nil {
		return backendGroups, errors.Wrap(err, "region.GetLoadbalancerPools()")
	}
	for i := 0; i < len(pools); i++ {
		backendGroups = append(backendGroups, &pools[i])
	}
	return backendGroups, nil
}

func (region *SRegion) GetClient() *SOpenStackClient {
	return region.client
}

func (region *SRegion) GetId() string {
	return region.Name
}

func (region *SRegion) GetName() string {
	return fmt.Sprintf("%s-%s", region.client.cpcfg.Name, region.Name)
}

func (region *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(region.GetName()).CN(region.GetName())
	return table
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s/%s", CLOUD_PROVIDER_OPENSTACK, region.client.cpcfg.Id, region.Name)
}

func (region *SRegion) IsEmulated() bool {
	return false
}

func (region *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_OPENSTACK
}

func (region *SRegion) GetCloudEnv() string {
	return ""
}

func (region *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (region *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (region *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (region *SRegion) GetMaxVersion(service string) (string, error) {
	return region.client.GetMaxVersion(region.Name, service)
}

func (region *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	vpc, err := region.CreateVpc(opts.NAME, opts.Desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateVp")
	}
	return vpc, nil
}

func (region *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i++ {
		ihost, err := izones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i++ {
		istore, err := izones[i].GetIStorageById(id)
		if err == nil {
			return istore, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	iHosts := []cloudprovider.ICloudHost{}
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i++ {
		iZoneHost, err := izones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		iHosts = append(iHosts, iZoneHost...)
	}
	return iHosts, nil
}

func (region *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	iStorages := []cloudprovider.ICloudStorage{}
	for i := 0; i < len(izones); i++ {
		iZoneStores, err := izones[i].GetIStorages()
		if err != nil {
			return nil, err
		}
		iStorages = append(iStorages, iZoneStores...)
	}
	return iStorages, nil
}

func (region *SRegion) getStoragecache() *SStoragecache {
	if region.storageCache == nil {
		region.storageCache = &SStoragecache{region: region}
	}
	return region.storageCache
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := region.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return region.storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := region.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}

func (region *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	instance, err := region.GetInstance(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstance(%s)", id)
	}
	hosts, err := region.GetIHosts()
	if err != nil {
		return nil, err
	}
	for i := range hosts {
		host := hosts[i].(*SHypervisor)
		if instance.HypervisorHostname == host.HypervisorHostname {
			instance.host = host
		}
	}
	return instance, nil
}

func (region *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := region.GetDisk(id)
	if err != nil {
		_, err := region.GetInstance(id)
		if err == nil {
			return &SNovaDisk{region: region, instanceId: id}, nil
		}
		return nil, errors.Wrapf(err, "GetDisk(%s)", id)
	}
	return disk, nil
}

func (region *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := region.GetVpc(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc(%s)", id)
	}
	return vpc, nil
}

func (region *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := region.GetIZones()
	if err != nil {
		return nil, errors.Wrap(err, "GetIZones")
	}
	for i := 0; i < len(izones); i++ {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetZones() ([]SZone, error) {
	zones, err := region.getZones()
	if err != nil {
		return nil, errors.Wrap(err, "getZones")
	}
	ret := []SZone{}
	for i := 0; i < len(zones); i++ {
		if zones[i].ZoneName == "internal" {
			continue
		}
		zones[i].region = region
		ret = append(ret, zones[i])
	}
	return ret, nil
}

func (region *SRegion) fetchVpcs() error {
	if len(region.vpcs) > 0 {
		return nil
	}
	var err error
	region.vpcs, err = region.GetVpcs("")
	if err != nil {
		return errors.Wrap(err, "GetVpcs")
	}
	return nil
}

func (region *SRegion) ecsList(resource string, query url.Values) (jsonutils.JSONObject, error) {
	return region.client.ecsRequest(region.Name, httputils.GET, resource, query, nil)
}

func (region *SRegion) ecsGet(resource string) (jsonutils.JSONObject, error) {
	return region.client.ecsRequest(region.Name, httputils.GET, resource, nil, nil)
}

func (region *SRegion) ecsUpdate(resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.ecsRequest(region.Name, httputils.PUT, resource, nil, params)
}

func (region *SRegion) ecsPost(resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.ecsRequest(region.Name, httputils.POST, resource, nil, params)
}

func (region *SRegion) ecsDo(projectId, resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.ecsDo(projectId, region.Name, resource, params)
}

func (region *SRegion) ecsDelete(resource string) (jsonutils.JSONObject, error) {
	return region.client.ecsRequest(region.Name, httputils.DELETE, resource, nil, nil)
}

func (region *SRegion) ecsCreate(projectId, resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.ecsCreate(projectId, region.Name, resource, params)
}

func (region *SRegion) vpcList(resource string, query url.Values) (jsonutils.JSONObject, error) {
	return region.client.vpcRequest(region.Name, httputils.GET, resource, query, nil)
}

func (region *SRegion) vpcGet(resource string) (jsonutils.JSONObject, error) {
	return region.client.vpcRequest(region.Name, httputils.GET, resource, nil, nil)
}

func (region *SRegion) vpcUpdate(resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.vpcRequest(region.Name, httputils.PUT, resource, nil, params)
}

func (region *SRegion) vpcPost(resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.vpcRequest(region.Name, httputils.POST, resource, nil, params)
}

func (region *SRegion) vpcDelete(resource string) (jsonutils.JSONObject, error) {
	return region.client.vpcRequest(region.Name, httputils.DELETE, resource, nil, nil)
}

func (region *SRegion) imageList(resource string, query url.Values) (jsonutils.JSONObject, error) {
	return region.client.imageRequest(region.Name, httputils.GET, resource, query, nil)
}

func (region *SRegion) imageGet(resource string) (jsonutils.JSONObject, error) {
	return region.client.imageRequest(region.Name, httputils.GET, resource, nil, nil)
}

func (region *SRegion) imagePost(resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.imageRequest(region.Name, httputils.POST, resource, nil, params)
}

func (region *SRegion) imageDelete(resource string) (jsonutils.JSONObject, error) {
	return region.client.imageRequest(region.Name, httputils.DELETE, resource, nil, nil)
}

func (region *SRegion) imageUpload(url string, size int64, body io.Reader, callback func(progress float32)) error {
	resp, err := region.client.imageUpload(region.Name, url, size, body, callback)
	_, _, err = httputils.ParseResponse("", resp, err, region.client.debug)
	return err
}

// Block Storage
func (region *SRegion) bsList(resource string, query url.Values) (jsonutils.JSONObject, error) {
	return region.client.bsRequest(region.Name, httputils.GET, resource, query, nil)
}

func (region *SRegion) bsGet(resource string) (jsonutils.JSONObject, error) {
	return region.client.bsRequest(region.Name, httputils.GET, resource, nil, nil)
}

func (region *SRegion) bsUpdate(resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.bsRequest(region.Name, httputils.PUT, resource, nil, params)
}

func (region *SRegion) bsPost(resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.bsRequest(region.Name, httputils.POST, resource, nil, params)
}

func (region *SRegion) bsDelete(resource string) (jsonutils.JSONObject, error) {
	return region.client.bsRequest(region.Name, httputils.DELETE, resource, nil, nil)
}

func (region *SRegion) bsCreate(projectId, resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.bsCreate(projectId, region.Name, resource, params)
}

//loadbalancer

func (region *SRegion) lbList(resource string, query url.Values) (jsonutils.JSONObject, error) {
	return region.client.lbRequest(region.Name, httputils.GET, resource, query, nil)
}

func (region *SRegion) lbGet(resource string) (jsonutils.JSONObject, error) {
	return region.client.lbRequest(region.Name, httputils.GET, resource, nil, nil)
}

func (region *SRegion) lbUpdate(resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.lbRequest(region.Name, httputils.PUT, resource, nil, params)
}

func (region *SRegion) lbPost(resource string, params interface{}) (jsonutils.JSONObject, error) {
	return region.client.lbRequest(region.Name, httputils.POST, resource, nil, params)
}

func (region *SRegion) lbDelete(resource string) (jsonutils.JSONObject, error) {
	return region.client.lbRequest(region.Name, httputils.DELETE, resource, nil, nil)
}

func (region *SRegion) ProjectId() string {
	return region.client.tokenCredential.GetProjectId()
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	zones, err := region.GetZones()
	if err != nil {
		return nil, errors.Wrapf(err, "GetZones")
	}
	ret := []cloudprovider.ICloudZone{}
	for i := range zones {
		ret = append(ret, &zones[i])
	}
	return ret, nil
}

func (region *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	err := region.fetchVpcs()
	if err != nil {
		return nil, errors.Wrap(err, "fetchVpcs")
	}
	ivpcs := []cloudprovider.ICloudVpc{}
	for i := range region.vpcs {
		region.vpcs[i].region = region
		ivpcs = append(ivpcs, &region.vpcs[i])
	}
	return ivpcs, nil
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := region.GetEips("")
	if err != nil {
		return nil, err
	}
	ieips := []cloudprovider.ICloudEIP{}
	for i := 0; i < len(eips); i++ {
		eips[i].region = region
		ieips = append(ieips, &eips[i])
	}
	return ieips, nil
}

func (region *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	network, err := region.GetNetwork(eip.NetworkExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetwork(%s)", eip.NetworkExternalId)
	}
	ieip, err := region.CreateEip(network.NetworkId, eip.NetworkExternalId, eip.IP, eip.ProjectId)
	if err != nil {
		return nil, errors.Wrap(err, "CreateEip")
	}
	return ieip, nil
}

func (region *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	return region.GetEip(eipId)
}

func (region *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	loadbalancers := []cloudprovider.ICloudLoadbalancer{}
	sloadbalancers, err := region.GetLoadbalancers()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetLoadbalancers()")
	}
	for i := 0; i < len(sloadbalancers); i++ {
		loadbalancers = append(loadbalancers, &sloadbalancers[i])
	}
	return loadbalancers, nil
}

func (region *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	sloadbalancer, err := region.GetLoadbalancerbyId(loadbalancerId)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetLoadbalancerbyId(%s)", loadbalancerId)
	}
	return sloadbalancer, nil
}

func (region *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return region.GetLoadbalancerAclDetail(aclId)
}

func (region *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	iloadbalancerAcls := []cloudprovider.ICloudLoadbalancerAcl{}
	acls, err := region.GetLoadBalancerAcls()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetLoadBalancerAcls")
	}
	for i := 0; i < len(acls); i++ {
		iloadbalancerAcls = append(iloadbalancerAcls, &acls[i])
	}
	return iloadbalancerAcls, nil
}

func (region *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancerCreateOptions) (cloudprovider.ICloudLoadbalancer, error) {
	sloadbalancer, err := region.CreateLoadBalancer(loadbalancer)
	if err != nil {
		return nil, errors.Wrap(err, "region.CreateLoadBalancer")
	}
	return sloadbalancer, nil
}

func (region *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	sacl, err := region.CreateLoadBalancerAcl(acl)
	if err != nil {
		return nil, errors.Wrap(err, "region.CreateLoadBalancerAcl(acl)")
	}
	err = cloudprovider.WaitMultiStatus(sacl.listener, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_UNKNOWN}, 10*time.Second, 8*time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "cloudprovider.WaitMultiStatus")
	}
	if sacl.listener.GetStatus() == api.LB_STATUS_UNKNOWN {
		return nil, errors.Wrap(fmt.Errorf("status error"), "check status")
	}
	return sacl, nil
}

func (region *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	flavors, err := region.GetFlavors()
	if err != nil {
		return nil, err
	}
	iskus := make([]cloudprovider.ICloudSku, len(flavors))
	for i := 0; i < len(flavors); i++ {
		flavors[i].region = region
		iskus[i] = &flavors[i]
	}
	return iskus, nil
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, acl string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) DeleteIBucket(name string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	return false, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return region.GetSecurityGroup(secgroupId)
}

func (region *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	secgroups, err := region.GetSecurityGroups(opts.ProjectId, opts.Name)
	if err != nil {
		return nil, err
	}
	if len(secgroups) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(secgroups) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	secgroups[0].region = region
	return &secgroups[0], nil
}

func (region *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return region.CreateSecurityGroup(conf.ProjectId, conf.Name, conf.Desc)
}

func (region *SRegion) GetCapabilities() []string {
	return region.client.GetCapabilities()
}

func (region *SRegion) GetRouters() ([]SRouter, error) {
	resource := "/v2.0/routers"
	resp, err := region.vpcList(resource, nil)
	if err != nil {
		return nil, errors.Wrap(err, "vpcList.routes")
	}
	routers := []SRouter{}
	err = resp.Unmarshal(&routers, "routers")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	for i := 0; i < len(routers); i++ {
		ports, err := region.GetPorts("", routers[i].Id)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.region.GetPortsByDeviceId")
		}
		routers[i].ports = ports
	}
	return routers, nil
}

func (region *SRegion) fetchrouters() error {
	if len(region.routers) > 0 {
		return nil
	}
	routers, err := region.GetRouters()
	if err != nil {
		return errors.Wrap(err, "region.GetRouters()")
	}
	region.routers = routers
	return nil
}
