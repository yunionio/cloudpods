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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion
	client *SGoogleClient

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

func (region *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (region *SRegion) GetName() string {
	if name, ok := RegionNames[region.Name]; ok {
		return fmt.Sprintf("%s %s", CLOUD_PROVIDER_GOOGLE_CN, name)
	}
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_GOOGLE_CN, region.Name)
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

func (region *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_GOOGLE
}

func (region *SRegion) GetStatus() string {
	if region.Status == "UP" {
		return api.CLOUD_REGION_STATUS_INSERVER
	}
	return api.CLOUD_REGION_STATUS_OUTOFSERVICE
}

func (region *SRegion) IsEmulated() bool {
	return false
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
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

func (region *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	globalnetworks, err := region.client.fetchGlobalNetwork()
	if err != nil {
		return nil, errors.Wrap(err, "fetchGlobalNetwork")
	}
	ivpcs := []cloudprovider.ICloudVpc{}
	for i := range globalnetworks {
		vpc := SVpc{region: region, globalnetwork: &globalnetworks[i]}
		ivpcs = append(ivpcs, &vpc)
	}
	return ivpcs, nil
}

func (region *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := region.GetIVpcs()
	if err != nil {
		return nil, err
	}
	for i := range ivpcs {
		if ivpcs[i].GetGlobalId() == id {
			return ivpcs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	globalnetwork, err := region.CreateGlobalNetwork(name, desc)
	if err != nil {
		return nil, errors.Wrap(err, "region.CreateGlobalNetwork")
	}
	vpc := &SVpc{region: region, globalnetwork: globalnetwork}
	return vpc, nil
}

func (region *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
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

func (region *SRegion) GetProjectId() string {
	return region.client.projectId
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
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
	ivm, err := region.GetInstance(id)
	if err != nil {
		return nil, err
	}
	return ivm, nil
}

func (region *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	return disk, nil
}

func (region *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
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
	snapshot, err := region.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
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
	return region.client.ecsListAll(resource, params, retval)
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

func (region *SRegion) Get(id string, retval interface{}) error {
	return region.client.ecsGet(id, retval)
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

func (region *SRegion) StorageDo(id string, action string, params map[string]string, body jsonutils.JSONObject) error {
	opId, err := region.client.storageDo(id, action, params, body)
	if err != nil {
		return err
	}
	if strings.Index(opId, "/operations/") > 0 {
		_, err = region.WaitOperation(opId, id, action)
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
		_, err = region.WaitOperation(opId, id, action)
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
		_, err = region.WaitOperation(opId, id, action)
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
	_, err = region.WaitOperation(operation.SelfLink, id, "delete")
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

func (region *SRegion) Insert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	operation := &SOperation{}
	err := region.client.ecsInsert(resource, body, operation)
	if err != nil {
		return err
	}
	resourceId, err := region.WaitOperation(operation.SelfLink, resource, "insert")
	if err != nil {
		return errors.Wrapf(err, "region.WaitOperation(%s)", operation.SelfLink)
	}
	return region.Get(resourceId, retval)
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
	policies, err := region.fetchResourcePolicies()
	if err != nil {
		return nil, err
	}
	ipolicies := []cloudprovider.ICloudSnapshotPolicy{}
	for i := range policies {
		policies[i].region = region
		if strings.Contains(region.Name, policies[i].SnapshotSchedulePolicy.SnapshotProperties.StorageLocations[0]) {
			ipolicies = append(ipolicies, &policies[i])
		}
	}
	return ipolicies, nil
}

func (region *SRegion) GetISnapshotPolicyById(id string) (cloudprovider.ICloudSnapshotPolicy, error) {
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
	eip, err := region.CreateEip(args.Name, "")
	if err != nil {
		return nil, err
	}
	return eip, nil
}
