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

package google

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRegion struct {
	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion
	client *SGoogleClient

	capabilities []string
	Quotas       []SQuota

	Description       string
	ID                string
	Kind              string
	Name              string
	Status            string
	SelfLink          string
	CreationTimestamp time.Time
}

func (region *SRegion) GetClient() *SGoogleClient {
	return region.client
}

func (region *SRegion) GetName() string {
	if name, ok := RegionNames[region.Name]; ok {
		return fmt.Sprintf("%s %s", CLOUD_PROVIDER_GOOGLE_CN, name)
	}
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_GOOGLE_CN, region.Name)
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	en := fmt.Sprintf("%s %s", CLOUD_PROVIDER_GOOGLE, self.Name)
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName()).EN(en)
	return table
}

func (region *SRegion) GetId() string {
	return region.Name
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_GOOGLE, region.Name)
}

func (region *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if geoInfo, ok := LatitudeAndLongitude[region.Name]; ok {
		return geoInfo
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetCreatedAt() time.Time {
	return self.CreationTimestamp
}

func (region *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_GOOGLE
}

func (region *SRegion) GetStatus() string {
	if region.Status == "UP" || utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return api.CLOUD_REGION_STATUS_INSERVER
	}
	return api.CLOUD_REGION_STATUS_OUTOFSERVICE
}

func (region *SRegion) CreateIBucket(name string, storageClassStr string, acl string) error {
	_, err := region.CreateBucket(name, storageClassStr, cloudprovider.TBucketACLType(acl))
	return err
}

func (region *SRegion) DeleteIBucket(name string) error {
	return region.DeleteBucket(name)
}

func (region *SRegion) IBucketExist(name string) (bool, error) {
	//{"error":{"code":403,"details":"200420163731-compute@developer.gserviceaccount.com does not have storage.buckets.get access to test."}}
	//{"error":{"code":404,"details":"Not Found"}}
	_, err := region.GetBucket(name)
	if err == nil {
		return true, nil
	}
	if errors.Cause(err) == cloudprovider.ErrNotFound || strings.Contains(err.Error(), "storage.buckets.get access") {
		return false, nil
	}
	return false, err
}

func (region *SRegion) GetIBucketById(id string) (cloudprovider.ICloudBucket, error) {
	return cloudprovider.GetIBucketById(region, id)
}

func (region *SRegion) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return region.GetIBucketById(name)
}

func (region *SRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	iBuckets, err := region.client.getIBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "getIBuckets")
	}
	ret := []cloudprovider.ICloudBucket{}
	for i := range iBuckets {
		if iBuckets[i].GetLocation() != region.GetId() {
			continue
		}
		ret = append(ret, iBuckets[i])
	}
	return ret, nil
}

func (region *SRegion) IsEmulated() bool {
	return false
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return []cloudprovider.ICloudZone{}, nil
	}
	zones, err := region.GetZones(region.Name, 0, "")
	if err != nil {
		return nil, err
	}
	izones := []cloudprovider.ICloudZone{}
	for i := 0; i < len(zones); i++ {
		zones[i].region = region
		izones = append(izones, &zones[i])
	}
	return izones, nil
}

func (region *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].GetGlobalId() == id {
			return zones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if utils.IsInStringArray(self.Name, MultiRegions) || utils.IsInStringArray(self.Name, DualRegions) {
		return []cloudprovider.ICloudVpc{}, nil
	}
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

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	vpc, err := self.GetVpc(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc")
	}
	return vpc, nil
}

func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	if utils.IsInStringArray(self.Name, MultiRegions) || utils.IsInStringArray(self.Name, DualRegions) {
		return nil, cloudprovider.ErrNotSupported
	}
	gvpc, err := self.client.GetGlobalNetwork(opts.GlobalVpcExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetGlobalNetwork")
	}
	return self.CreateVpc(opts.NAME, gvpc.SelfLink, opts.CIDR, opts.Desc)
}

func (region *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	storage, err := region.GetStorage(id)
	if err != nil {
		return nil, err
	}
	zone, err := region.GetZone(storage.Zone)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetZone(%s)", storage.Zone)
	}
	zone.region = region
	storage.zone = zone
	return storage, nil
}

func (region *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	izones, err := region.GetIZones()
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

func (region *SRegion) GetProjectId() string {
	return region.client.projectId
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return []cloudprovider.ICloudEIP{}, nil
	}

	eips, err := region.GetEips("", 0, "")
	if err != nil {
		return nil, err
	}
	ieips := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eips[i].region = region
		ieips = append(ieips, &eips[i])
	}
	return ieips, nil
}

func (region *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	ivm, err := region.GetInstance(id)
	if err != nil {
		return nil, err
	}
	zone, err := region.GetZone(ivm.Zone)
	if err != nil {
		return nil, errors.Wrapf(err, "GetZone(%s)", ivm.Zone)
	}
	ivm.host = &SHost{zone: zone}
	return ivm, nil
}

func (region *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	disk, err := region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func (region *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	eip, err := region.GetEip(id)
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (region *SRegion) fetchSnapshots() error {
	if len(region.client.snapshots) > 0 {
		return nil
	}
	region.client.snapshots = map[string][]SSnapshot{}
	snapshots, err := region.GetSnapshots("", 0, "")
	if err != nil {
		return err
	}
	regionNames := []string{}
	for _, region := range region.client.iregions {
		regionNames = append(regionNames, region.GetId())
	}
	for _, snapshot := range snapshots {
		for _, location := range snapshot.StorageLocations {
			_regionName := ""
			if utils.IsInStringArray(location, regionNames) {
				_regionName = location
			} else {
				for _, regionName := range regionNames {
					if strings.HasPrefix(regionName, location) {
						_regionName = regionName
						break
					}
				}
			}
			if len(_regionName) > 0 {
				if _, ok := region.client.snapshots[_regionName]; !ok {
					region.client.snapshots[_regionName] = []SSnapshot{}
				}
				region.client.snapshots[_regionName] = append(region.client.snapshots[_regionName], snapshot)
				break
			}

		}
	}
	return nil
}

func (region *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return []cloudprovider.ICloudSnapshot{}, nil
	}

	region.fetchSnapshots()
	isnapshots := []cloudprovider.ICloudSnapshot{}
	if snapshots, ok := region.client.snapshots[region.Name]; ok {
		for i := range snapshots {
			snapshots[i].region = region
			isnapshots = append(isnapshots, &snapshots[i])
		}
	}
	return isnapshots, nil
}

func (region *SRegion) GetISnapshotById(id string) (cloudprovider.ICloudSnapshot, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	snapshot, err := region.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (region *SRegion) rdsDelete(id string) error {
	operation := &SOperation{}
	err := region.client.rdsDelete(id, operation)
	if err != nil {
		return errors.Wrap(err, "client.rdsDelete")
	}
	_, err = region.WaitRdsOperation(operation.SelfLink, id, "delete")
	if err != nil {
		return errors.Wrapf(err, "region.WaitRdsOperation(%s)", operation.SelfLink)
	}
	return nil
}

func (region *SRegion) rdsDo(id string, action string, params map[string]string, body jsonutils.JSONObject) error {
	opId, err := region.client.rdsDo(id, action, params, body)
	if err != nil {
		return err
	}
	if strings.Index(opId, "/operations/") > 0 {
		_, err = region.WaitRdsOperation(opId, id, action)
		return err
	}
	return nil
}

func (region *SRegion) rdsPatch(id string, body jsonutils.JSONObject) error {
	opId, err := region.client.rdsPatch(id, body)
	if err != nil {
		return err
	}
	if strings.Index(opId, "/operations/") > 0 {
		_, err = region.WaitRdsOperation(opId, id, "update")
		return err
	}
	return nil
}

func (region *SRegion) rdsUpdate(id string, params map[string]string, body jsonutils.JSONObject) error {
	opId, err := region.client.rdsUpdate(id, params, body)
	if err != nil {
		return err
	}
	if strings.Index(opId, "/operations/") > 0 {
		_, err = region.WaitRdsOperation(opId, id, "update")
		return err
	}
	return nil
}

func (region *SRegion) rdsGet(resource string, retval interface{}) error {
	return region.client.rdsGet(resource, retval)
}

func (region *SRegion) rdsInsert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	operation := SOperation{}
	err := region.client.rdsInsert(resource, body, &operation)
	if err != nil {
		return errors.Wrap(err, "rdsInsert")
	}
	resourceId, err := region.WaitRdsOperation(operation.SelfLink, resource, "insert")
	if err != nil {
		return errors.Wrapf(err, "region.WaitRdsOperation(%s)", operation.SelfLink)
	}
	return region.rdsGet(resourceId, retval)
}

func (region *SRegion) RdsListAll(resource string, params map[string]string, retval interface{}) error {
	return region.client.rdsListAll(resource, params, retval)
}

func (region *SRegion) RdsList(resource string, params map[string]string, maxResults int, pageToken string, retval interface{}) error {
	if maxResults == 0 && len(pageToken) == 0 {
		return region.RdsListAll(resource, params, retval)
	}
	if params == nil {
		params = map[string]string{}
	}
	params["maxResults"] = fmt.Sprintf("%d", maxResults)
	params["pageToken"] = pageToken
	resp, err := region.client.rdsList(resource, params)
	if err != nil {
		return errors.Wrap(err, "billingList")
	}
	if resp.Contains("items") && retval != nil {
		err = resp.Unmarshal(retval, "items")
		if err != nil {
			return errors.Wrap(err, "resp.Unmarshal")
		}
	}
	return nil
}

func (region *SRegion) BillingList(resource string, params map[string]string, pageSize int, pageToken string, retval interface{}) error {
	if pageSize == 0 && len(pageToken) == 0 {
		return region.BillingListAll(resource, params, retval)
	}
	if params == nil {
		params = map[string]string{}
	}
	params["pageSize"] = fmt.Sprintf("%d", pageSize)
	params["pageToken"] = pageToken
	resp, err := region.client.billingList(resource, params)
	if err != nil {
		return errors.Wrap(err, "billingList")
	}
	if resp.Contains("skus") && retval != nil {
		err = resp.Unmarshal(retval, "skus")
		if err != nil {
			return errors.Wrap(err, "resp.Unmarshal")
		}
	}
	return nil
}

func (region *SRegion) BillingListAll(resource string, params map[string]string, retval interface{}) error {
	return region.client.billingListAll(resource, params, retval)
}

func (region *SRegion) ListAll(resource string, params map[string]string, retval interface{}) error {
	return region.listAll("GET", resource, params, retval)
}

func (region *SRegion) listAll(method string, resource string, params map[string]string, retval interface{}) error {
	return region.client._ecsListAll(method, resource, params, retval)
}

func (region *SRegion) List(resource string, params map[string]string, maxResults int, pageToken string, retval interface{}) error {
	if maxResults == 0 && len(pageToken) == 0 {
		return region.ListAll(resource, params, retval)
	}
	if params == nil {
		params = map[string]string{}
	}
	params["maxResults"] = fmt.Sprintf("%d", maxResults)
	params["pageToken"] = pageToken
	resp, err := region.client.ecsList(resource, params)
	if err != nil {
		return errors.Wrap(err, "ecsList")
	}
	if resp.Contains("items") && retval != nil {
		err = resp.Unmarshal(retval, "items")
		if err != nil {
			return errors.Wrap(err, "resp.Unmarshal")
		}
	}
	return nil
}

func (region *SRegion) Get(resourceType, id string, retval interface{}) error {
	return region.client.ecsGet(resourceType, id, retval)
}

func (self *SGoogleClient) GetBySelfId(id string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "GET", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, id, nil, nil, self.debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SRegion) GetBySelfId(id string, retval interface{}) error {
	return self.client.GetBySelfId(id, retval)
}

func (region *SRegion) StorageListAll(resource string, params map[string]string, retval interface{}) error {
	return region.client.storageListAll(resource, params, retval)
}

func (region *SRegion) StorageList(resource string, params map[string]string, maxResults int, pageToken string, retval interface{}) error {
	if maxResults == 0 && len(pageToken) == 0 {
		return region.client.storageListAll(resource, params, retval)
	}
	if params == nil {
		params = map[string]string{}
	}
	params["maxResults"] = fmt.Sprintf("%d", maxResults)
	params["pageToken"] = pageToken
	resp, err := region.client.storageList(resource, params)
	if err != nil {
		return errors.Wrap(err, "storageList")
	}
	if resp.Contains("items") && retval != nil {
		err = resp.Unmarshal(retval, "items")
		if err != nil {
			return errors.Wrap(err, "resp.Unmarshal")
		}
	}
	return nil
}

func (region *SRegion) StorageGet(id string, retval interface{}) error {
	return region.client.storageGet(id, retval)
}

func (region *SRegion) StoragePut(id string, body jsonutils.JSONObject, retval interface{}) error {
	return region.client.storagePut(id, body, retval)
}

func (region *SRegion) StorageDo(id string, action string, params map[string]string, body jsonutils.JSONObject) error {
	opId, err := region.client.storageDo(id, action, params, body)
	if err != nil {
		return err
	}
	if strings.Index(opId, "/operations/") > 0 {
		_, err = region.client.WaitOperation(opId, id, action)
		return err
	}
	return nil
}

func (region *SRegion) Do(id string, action string, params map[string]string, body jsonutils.JSONObject) error {
	opId, err := region.client.ecsDo(id, action, params, body)
	if err != nil {
		return err
	}
	if strings.Index(opId, "/operations/") > 0 {
		_, err = region.client.WaitOperation(opId, id, action)
		return err
	}
	return nil
}

func (region *SRegion) Patch(id string, action string, params map[string]string, body jsonutils.JSONObject) error {
	opId, err := region.client.ecsPatch(id, action, params, body)
	if err != nil {
		return err
	}
	if strings.Index(opId, "/operations/") > 0 {
		_, err = region.client.WaitOperation(opId, id, action)
		return err
	}
	return nil
}

func (region *SRegion) StorageDelete(id string) error {
	return region.client.storageDelete(id, nil)
}

func (region *SRegion) Delete(id string) error {
	operation := &SOperation{}
	err := region.client.ecsDelete(id, operation)
	if err != nil {
		return errors.Wrap(err, "client.ecsDelete")
	}
	_, err = region.client.WaitOperation(operation.SelfLink, id, "delete")
	if err != nil {
		return errors.Wrapf(err, "region.WaitOperation(%s)", operation.SelfLink)
	}
	return nil
}

func (region *SRegion) StorageInsert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	return region.client.storageInsert(resource, body, retval)
}

func (region *SRegion) CloudbuildInsert(body jsonutils.JSONObject) error {
	result := &struct {
		Name string
	}{}
	resource := fmt.Sprintf("projects/%s/builds", region.GetProjectId())
	err := region.client.cloudbuildInsert(resource, body, result)
	if err != nil {
		return errors.Wrap(err, "insert")
	}
	err = cloudprovider.Wait(time.Second*10, time.Minute*40, func() (bool, error) {
		operation, err := region.GetCloudbuildOperation(result.Name)
		if err != nil {
			return false, errors.Wrapf(err, "region.GetCloudbuildOperation(%s)", result.Name)
		}
		status := operation.Metadata.Build.Status
		log.Debugf("cloudbuild %s status: %s", result.Name, status)
		if status == "FAILURE" {
			return false, fmt.Errorf("cloudbuild failed error log: %s", operation.Metadata.Build.LogUrl)
		}
		if status == "SUCCESS" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "cloudprovider.Wait")
	}
	return nil
}

func (region *SRegion) cloudbuildGet(id string, retval interface{}) error {
	return region.client.cloudbuildGet(id, retval)
}

func (self *SGoogleClient) Insert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	operation := &SOperation{}
	err := self.ecsInsert(resource, body, operation)
	if err != nil {
		return err
	}
	resourceId, err := self.WaitOperation(operation.SelfLink, resource, "insert")
	if err != nil {
		return errors.Wrapf(err, "region.WaitOperation(%s)", operation.SelfLink)
	}
	return self.GetBySelfId(resourceId, retval)
}

func (self *SRegion) Insert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	return self.client.Insert(resource, body, retval)
}

func (region *SRegion) fetchResourcePolicies() ([]SResourcePolicy, error) {
	if len(region.client.resourcepolices) > 0 {
		return region.client.resourcepolices, nil
	}
	policies, err := region.GetResourcePolicies(0, "")
	if err != nil {
		return nil, err
	}
	region.client.resourcepolices = policies
	return policies, nil
}

func (region *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return []cloudprovider.ICloudSnapshotPolicy{}, nil
	}

	policies, err := region.fetchResourcePolicies()
	if err != nil {
		return nil, err
	}
	ipolicies := []cloudprovider.ICloudSnapshotPolicy{}
	for i := range policies {
		policies[i].region = region
		if utils.IsInStringArray(region.Name, policies[i].SnapshotSchedulePolicy.SnapshotProperties.StorageLocations) {
			ipolicies = append(ipolicies, &policies[i])
		}
	}
	return ipolicies, nil
}

func (region *SRegion) GetISnapshotPolicyById(id string) (cloudprovider.ICloudSnapshotPolicy, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	policy, err := region.GetResourcePolicy(id)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(region.Name, policy.SnapshotSchedulePolicy.SnapshotProperties.StorageLocations[0]) {
		return nil, cloudprovider.ErrNotFound
	}
	return policy, nil
}

func (region *SRegion) CreateEIP(args *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotSupported
	}

	eip, err := region.CreateEip(args.Name, "")
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (region *SRegion) GetCapabilities() []string {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return []string{cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE}
	}
	if region.capabilities == nil {
		return region.client.GetCapabilities()
	}
	return region.capabilities
}

func (region *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return []cloudprovider.ICloudDBInstance{}, nil
	}

	instances, err := region.GetDBInstances(0, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstances")
	}
	ret := []cloudprovider.ICloudDBInstance{}
	for i := range instances {
		instances[i].region = region
		ret = append(ret, &instances[i])
	}
	return ret, nil
}

func (region *SRegion) GetIDBInstanceById(instanceId string) (cloudprovider.ICloudDBInstance, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	instance, err := region.GetDBInstance(instanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstance(%s)", instanceId)
	}
	return instance, nil
}

func (region *SRegion) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return []cloudprovider.ICloudDBInstanceBackup{}, nil
	}

	instances, err := region.GetDBInstances(0, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstances")
	}
	ret := []cloudprovider.ICloudDBInstanceBackup{}
	for i := range instances {
		instances[i].region = region
		backups, err := region.GetDBInstanceBackups(instances[i].Name)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDBInstanceBackups(%s)", instances[i].Name)
		}
		for j := range backups {
			backups[j].rds = &instances[i]
			ret = append(ret, &backups[j])
		}
	}
	return ret, nil
}

func (region *SRegion) GetIDBInstanceBackupById(backupId string) (cloudprovider.ICloudDBInstanceBackup, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotFound
	}

	backup, err := region.GetDBInstanceBackup(backupId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceBackup(%s)", backupId)
	}
	return backup, nil
}

func (region *SRegion) CreateIDBInstance(desc *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	if utils.IsInStringArray(region.Name, MultiRegions) || utils.IsInStringArray(region.Name, DualRegions) {
		return nil, cloudprovider.ErrNotSupported
	}

	rds, err := region.CreateDBInstance(desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateDBInstance")
	}
	return rds, nil
}

func (region *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := region.GetInstances("", 0, "")
	if err != nil {
		return nil, err
	}
	iVMs := []cloudprovider.ICloudVM{}
	for i := range instances {
		iVMs = append(iVMs, &instances[i])
	}
	return iVMs, nil
}