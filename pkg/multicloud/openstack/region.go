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
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SRegion struct {
	multicloud.SRegion

	client *SOpenStackClient

	Name string

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache
}

func (region *SRegion) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetClient() *SOpenStackClient {
	return region.client
}

func (region *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (region *SRegion) GetId() string {
	return region.Name
}

func (region *SRegion) GetName() string {
	return fmt.Sprintf("%s-%s", region.client.providerName, region.Name)
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s/%s", CLOUD_PROVIDER_OPENSTACK, region.client.providerID, region.Name)
}

func (region *SRegion) IsEmulated() bool {
	return false
}

func (region *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_OPENSTACK
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

func (region *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	params := map[string]map[string]string{
		"network": {
			"name":        name,
			"description": desc,
		},
	}
	_, resp, err := region.Post("network", "/v2.0/networks", "", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	err = region.fetchInfrastructure()
	if err != nil {
		return nil, err
	}
	vpcId, err := resp.GetString("network", "id")
	if err != nil {
		return nil, err
	}
	return region.GetIVpcById(vpcId)
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
		} else if err != cloudprovider.ErrNotFound {
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
		} else if err != cloudprovider.ErrNotFound {
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
	return region.GetInstance(id)
}

func (region *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return region.GetDisk(id)
}

func (region *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := region.GetIVpcs()
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

func (region *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i++ {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) fetchZones() error {
	zone := &SZone{region: region, ZoneName: region.Name, cachedHosts: map[string][]string{}}
	_, resp, err := region.List("compute", "/os-availability-zone/detail", "", jsonutils.NewDict())
	if err != nil {
		return err
	}
	zones := []SZone{}
	if err := resp.Unmarshal(&zones, "availabilityZoneInfo"); err != nil {
		return err
	}
	for i := 0; i < len(zones); i++ {
		if zones[i].ZoneName == "internal" {
			continue
		}
		zone.cachedHosts[zones[i].ZoneName] = []string{}
		for hostname, hostInfo := range zones[i].Hosts {
			for k := range hostInfo {
				if k == "nova-compute" {
					zone.cachedHosts[zones[i].ZoneName] = append(zone.cachedHosts[zones[i].ZoneName], hostname)
				}
			}
		}
	}
	region.izones = []cloudprovider.ICloudZone{zone}
	return nil
}

func (region *SRegion) fetchIVpcs() error {
	vpcs, err := region.GetVpcs()
	if err != nil {
		return err
	}
	region.ivpcs = []cloudprovider.ICloudVpc{}
	for i := 0; i < len(vpcs); i++ {
		vpcs[i].region = region
		region.ivpcs = append(region.ivpcs, &vpcs[i])
	}
	return nil
}

func (region *SRegion) fetchInfrastructure() error {
	if len(region.izones) == 0 {
		if err := region.fetchZones(); err != nil {
			return err
		}
	}
	if err := region.fetchIVpcs(); err != nil {
		return err
	}
	for i := 0; i < len(region.ivpcs); i++ {
		for j := 0; j < len(region.izones); j++ {
			zone := region.izones[j].(*SZone)
			vpc := region.ivpcs[i].(*SVpc)
			wire := SWire{zone: zone, vpc: vpc}
			zone.addWire(&wire)
			vpc.addWire(&wire)
		}
	}
	return nil
}

func (region *SRegion) Get(service, url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	if strings.HasSuffix(url, "/") {
		return nil, nil, cloudprovider.ErrNotFound
	}
	header, resp, err := region.client.Request(region.Name, service, "GET", url, microversion, body)
	if err != nil {
		if jsonErr, ok := err.(*httputils.JSONClientError); ok {
			if jsonErr.Code == 404 || strings.HasSuffix(jsonErr.Class, "NotFound") {
				return nil, nil, cloudprovider.ErrNotFound
			}
		}
		return nil, nil, err
	}
	return header, resp, nil
}

func (region *SRegion) List(service, url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	header, resp, err := region.client.Request(region.Name, service, "GET", url, microversion, body)
	if err != nil {
		if jsonErr, ok := err.(*httputils.JSONClientError); ok {
			if jsonErr.Code == 404 || strings.HasSuffix(jsonErr.Class, "NotFound") {
				return nil, nil, cloudprovider.ErrNotFound
			}
		}
		return nil, nil, err
	}
	return header, resp, nil
}

func (region *SRegion) Post(service, url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return region.client.Request(region.Name, service, "POST", url, microversion, body)
}

func (region *SRegion) Update(service, url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return region.client.Request(region.Name, service, "PUT", url, microversion, body)
}

func (region *SRegion) Delete(service, url string, microversion string) (*http.Response, error) {
	return region.client.RawRequest(region.Name, service, "DELETE", url, microversion, nil)
}

func (region *SRegion) CinderList(url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	for _, service := range []string{"volumev3", "volumev2", "volume"} {
		header, resp, err := region.Get(service, url, microversion, body)
		if err == nil || !strings.Contains(err.Error(), "No such service") {
			return header, resp, err
		}
		log.Debugf("failed to list %s by service %s error: %v, try another", url, service, err)
	}
	return nil, nil, fmt.Errorf("failed to get %s by cinder service", url)
}

func (region *SRegion) CinderGet(url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	if strings.HasSuffix(url, "/") {
		return nil, nil, cloudprovider.ErrNotFound
	}
	for _, service := range []string{"volumev3", "volumev2", "volume"} {
		header, resp, err := region.Get(service, url, microversion, body)
		if err == nil || err == cloudprovider.ErrNotFound || !strings.Contains(err.Error(), "No such service") {
			return header, resp, err
		}
		log.Debugf("failed to get %s by service %s error: %v, try another", url, service, err)
	}
	return nil, nil, fmt.Errorf("failed to get %s by cinder service", url)
}

func (region *SRegion) CinderCreate(url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	for _, service := range []string{"volumev3", "volumev2", "volume"} {
		header, resp, err := region.Post(service, url, microversion, body)
		if err == nil || !strings.Contains(err.Error(), "No such service") {
			return header, resp, err
		}
		log.Debugf("failed to create %s by service %s error: %v, try another", url, service, err)
	}
	return nil, nil, fmt.Errorf("failed to create %s by cinder service", url)
}

func (region *SRegion) CinderDelete(url string, microversion string) (*http.Response, error) {
	if strings.HasSuffix(url, "/") {
		return nil, cloudprovider.ErrNotFound
	}
	for _, service := range []string{"volumev3", "volumev2", "volume"} {
		resp, err := region.Delete(service, url, microversion)
		if err == nil || !strings.Contains(err.Error(), "No such service") {
			return resp, err
		}
		log.Debugf("failed to delete %s by service %s error: %v, try another", url, service, err)
	}
	return nil, fmt.Errorf("failed to delete %s by cinder service", url)
}

func (region *SRegion) CinderAction(url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	for _, service := range []string{"volumev3", "volumev2", "volume"} {
		header, resp, err := region.Post(service, url, microversion, body)
		if err == nil || !strings.Contains(err.Error(), "No such service") {
			return header, resp, err
		}
		log.Debugf("failed to operate %s by service %s error: %v, try another", url, service, err)
	}
	return nil, nil, fmt.Errorf("failed to operate %s by cinder service", url)
}

func (region *SRegion) ProjectId() string {
	return region.client.tokenCredential.GetProjectId()
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if region.izones == nil {
		if err := region.fetchInfrastructure(); err != nil {
			return nil, err
		}
	}
	return region.izones, nil
}

func (region *SRegion) GetVersion(service string) (string, string, error) {
	return region.client.getVersion(region.Name, service)
}

func (region *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if err := region.fetchInfrastructure(); err != nil {
		return nil, err
	}
	return region.ivpcs, nil
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := region.GetEips()
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
	return region.CreateEip(eip)
}

func (region *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	return region.GetEip(eipId)
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

func (region *SRegion) GetISkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
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

func (region *SRegion) DeleteISkuByName(name string) error {
	skus, err := region.GetISkus("")
	if err != nil {
		return errors.Wrap(err, "region.GetISkus()")
	}
	for _, sku := range skus {
		if sku.GetName() == name {
			err = sku.Delete()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (region *SRegion) GetISkuById(skuId string) (cloudprovider.ICloudSku, error) {
	return region.GetFlavor(skuId)
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

func (region *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return region.CreateSecurityGroup(conf.Name, conf.Desc)
}
