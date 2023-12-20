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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SModelartsPool struct {
	region *SRegion
	multicloud.SResourceBase

	Metadata     SModelartsPoolMetadata `json:"metadata"`
	Spec         SModelartsPoolSpec     `json:"spec"`
	Status       SModelartsPoolStatus   `json:"status"`
	InstanceType string
	WorkType     string
}

type SModelartsPoolMetadata struct {
	Name              string `json:"name"`
	CreationTimestamp string `json:"creationTimestamp"`
	Labels            SModelartsPoolMeatadataLabel
	Annotations       SModelartsPoolMetadataAnnotations `json:"annotations"`
}

type SModelartsPoolMeatadataLabel struct {
	WorkspaceId string `json:"os.modelarts/workspace.id"`
	Name        string `json:"os.modelarts/name"`
	ResourceId  string `json:"os.modelarts/resource.id"`
}

type SModelartsPoolMetadataAnnotations struct {
	Describe          string `json:"os.modelarts/description"`
	BillingType       string `json:"os.modelarts/billing.mode"`
	BillingCycle      string `json:"os.modelarts/period.num"`
	BillingPeriodType string `json:"os.modelarts/period.type"`
	BillingMod        string `json:"os.modelarts/charging.mode"`
	BillingRenew      string `json:"os.modelarts/auto.renew"`
	OrderId           string `json:"os.modelarts/order.id"`
}

type SModelartsPoolSpec struct {
	Type     string                   `json:"type"`
	Scope    []string                 `json:"scope"`
	Resource []SModelartsPoolResource `json:"resources"`
}

type SModelartsPoolResource struct {
	Flavor string `json:"flavor"`
	Count  int    `json:"count"`
	cloudprovider.Azs
}

type SNodeStatus struct {
	Creating  []SNodeFlavor `json:"creating"`
	Available []SNodeFlavor `json:"available"`
	Abnormal  []SNodeFlavor `json:"abnormal"`
	Deleting  []SNodeFlavor `json:"deleting"`
}
type SNodeFlavor struct {
	Flavor string `json:"flavor"`
	Count  int    `json:"count"`
}

type SModelartsPoolStatus struct {
	Phase    string      `json:"phase"`
	Message  string      `json:"message"`
	Resource SNodeStatus `json:"resources"`
}

type SModelartsPoolNetwork struct {
	Metadata SModelartsPoolNetworkMetadata `json:"metadata"`
	Spec     SModelartsNetworkSpce         `json:"spec"`
	Status   SModelartsNetworkStatus       `json:"status"`
}

type SModelartsPoolNetworkMetadata struct {
	Name              string `json:"name"`
	CreationTimestamp string `json:"creationTimestamp"`
}

type SModelartsNetworkSpce struct {
	Cidr string `json:"cidr"`
}

type SModelartsNetworkStatus struct {
	Phase string `json:"phase"`
}

func (self *SRegion) GetIModelartsPools() ([]cloudprovider.ICloudModelartsPool, error) {
	pools := make([]SModelartsPool, 0)
	resObj, err := self.list(SERVICE_MODELARTS, "pools", nil)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetPools")
	}
	err = resObj.Unmarshal(&pools, "items")
	if err != nil {
		return nil, errors.Wrap(err, "resObj unmarshal")
	}
	res := make([]cloudprovider.ICloudModelartsPool, len(pools))
	for i := 0; i < len(pools); i++ {
		pools[i].region = self
		res[i] = &pools[i]
	}

	return res, nil
}

func (self *SRegion) CreateIModelartsPool(args *cloudprovider.ModelartsPoolCreateOption, callback func(id string)) (cloudprovider.ICloudModelartsPool, error) {
	if len(args.Cidr) == 0 {
		args.Cidr = "192.168.20.0/24"
	}
	netObj, err := self.list(SERVICE_MODELARTS, "network", nil)
	if err != nil {
		return nil, errors.Wrap(err, "SHuaweiClient.GetPools")
	}
	netRes := make([]SModelartsPoolNetwork, 0)
	netObj.Unmarshal(&netRes, "items")
	netId := ""
	for _, net := range netRes {
		if net.Spec.Cidr == args.Cidr {
			netId = net.Metadata.Name
		}
	}

	if len(netId) == 0 {
		createNetObj, err := self.client.CreatePoolNetworks(args.Cidr)
		if err != nil {
			return nil, errors.Wrap(err, "SHuaweiClient.CreatePoolNetworks")
		}
		netId, _ = createNetObj.GetString("metadata", "name")
		for i := 0; i < 10; i++ {
			netDetailObj, err := self.list(SERVICE_MODELARTS_V1, "networks/"+netId, nil)
			if err != nil {
				return nil, errors.Wrap(err, "SHuaweiClient.NetworkDetail")
			}
			netStatus, _ := netDetailObj.GetString("status", "phase")
			if netStatus == "Active" {
				break
			} else {
				time.Sleep(10 * time.Second)
			}
		}
	}

	scopeArr := strings.Split(args.WorkType, ",")
	params := map[string]interface{}{
		"apiVersion": "v2",
		"kind":       "Pool",
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				"os.modelarts/name":         args.Name,
				"os.modelarts/workspace.id": "0",
			},
		},
		"spec": map[string]interface{}{
			"type":  "Dedicate",
			"scope": scopeArr,
			"network": map[string]interface{}{
				"name": netId,
			},

			"resources": []map[string]interface{}{
				{
					"flavor": args.InstanceType,
					"count":  args.NodeCount,
				},
			},
		},
	}
	obj, err := self.post(SERVICE_MODELARTS, "pools", params)
	if err != nil {
		return nil, errors.Wrap(err, "SHuaweiClient.CreatePools")
	}
	pool := &SModelartsPool{
		region: self,
	}
	obj.Unmarshal(pool)
	if callback != nil {
		callback(pool.GetId())
	}
	// 对于新建后可能会存在一段时间list查不到
	time.Sleep(2 * time.Minute)
	return self.waitCreate(pool)
}

func (region *SRegion) waitCreate(pool *SModelartsPool) (cloudprovider.ICloudModelartsPool, error) {
	startTime := time.Now()
	for time.Since(startTime) < 2*time.Hour {
		pool.RefreshForCreate()
		if utils.IsInStringArray(pool.GetStatus(), []string{compute.MODELARTS_POOL_STATUS_RUNNING, compute.MODELARTS_POOL_STATUS_CREATE_FAILED}) {
			return pool, nil
		}
		time.Sleep(15 * time.Second)
	}
	return nil, errors.ErrTimeout
}

func (self *SRegion) DeletePool(poolName string) (jsonutils.JSONObject, error) {
	resource := fmt.Sprintf("pools/%s", poolName)
	return self.delete(SERVICE_MODELARTS, resource)
}

func (self *SRegion) GetIModelartsPoolById(poolId string) (cloudprovider.ICloudModelartsPool, error) {
	obj, err := self.list(SERVICE_MODELARTS, "pools/"+poolId, nil)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, "")
		}
		return nil, errors.Wrap(err, "region.modelartsPoolByName")
	}
	pool := &SModelartsPool{}
	obj.Unmarshal(&pool)
	res := []cloudprovider.ICloudModelartsPool{}
	for i := 0; i < 1; i++ {
		pool.region = self
		res = append(res, pool)
	}
	return res[0], nil
}

func (self *SRegion) MonitorPool(poolId string) (*SModelartsMetrics, error) {
	resource := fmt.Sprintf("pools/%s/monitor", poolId)
	resObj, err := self.list(SERVICE_MODELARTS, resource, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "send request error")
	}
	metrics := SModelartsMetrics{}
	err = resObj.Unmarshal(&metrics)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarsh error")
	}
	return &metrics, nil
}

type SModelartsMetrics struct {
	Metrics []SModelartsMetric `json:"metrics"`
}

type SModelartsMetric struct {
	Metric     SModelartsMetricInfo   `json:"metric"`
	Datapoints []SModelartsDataPoints `json:"dataPoints"`
}

type SModelartsMetricInfo struct {
	Dimensions []SModelartsDimensions `json:"dimensions"`
	MetricName string
	Namespace  string
}

type SModelartsDimensions struct {
	Name  string
	Value string
}

type SModelartsDataPoints struct {
	Timestamp  int64
	Unit       string
	Statistics []ModelartsStatistics
}

type ModelartsStatistics struct {
	Statistic string
	Value     float64
}

func (self *SHuaweiClient) GetPoolNetworks(poolName string) (jsonutils.JSONObject, error) {
	return self.list(SERVICE_MODELARTS_V1, self.clientRegion, "networks", nil)
}

func (self *SHuaweiClient) CreatePoolNetworks(cidr string) (jsonutils.JSONObject, error) {
	params := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Network",
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				"os.modelarts/name":         "test",
				"os.modelarts/workspace.id": "0",
			},
		},
		"spec": map[string]interface{}{
			"cidr": cidr,
		},
	}
	return self.post(SERVICE_MODELARTS_V1, self.clientRegion, "networks", params)
}

func (self *SModelartsPool) GetCreatedAt() time.Time {
	ret, _ := time.Parse("2006-01-02T15:04:05CST", self.Metadata.CreationTimestamp)
	if !ret.IsZero() {
		ret = ret.Add(time.Hour * 8)
	}
	return ret
}

func (self *SModelartsPool) GetGlobalId() string {
	return self.Metadata.Name
}

func (self *SModelartsPool) GetId() string {
	return self.Metadata.Name
}

func (self *SModelartsPool) GetName() string {
	return self.Metadata.Labels.Name
}

func (self *SModelartsPool) GetStatus() string {
	res := strings.ToLower(self.Status.Phase)
	availableCount := 0
	for _, node := range self.Status.Resource.Available {
		availableCount += node.Count
	}

	switch {
	case res == compute.MODELARTS_POOL_STATUS_RUNNING && availableCount == self.GetNodeCount():
		res = compute.MODELARTS_POOL_STATUS_RUNNING
	case res == compute.MODELARTS_POOL_STATUS_DELETING:
		res = compute.MODELARTS_POOL_STATUS_DELETING
	case res == compute.MODELARTS_POOL_STATUS_ERROR:
		res = compute.MODELARTS_POOL_STATUS_ERROR
	case (res == compute.MODELARTS_POOL_STATUS_RUNNING && len(self.Status.Resource.Creating) != 0) || res == compute.MODELARTS_POOL_STATUS_CREATING:
		res = compute.MODELARTS_POOL_STATUS_CREATING
	case self.Status.Phase == "CreationFailed":
		res = compute.MODELARTS_POOL_STATUS_CREATE_FAILED
	case self.Status.Phase == "SeclingFailed":
		res = compute.MODELARTS_POOL_STATUS_CHANGE_CONFIG_FAILED
	default:
		res = compute.MODELARTS_POOL_STATUS_UNKNOWN
	}
	return res
}

func (self *SModelartsPool) GetStatusMessage() string {
	return self.Status.Message
}

func (self *SModelartsPool) GetSysTags() map[string]string {
	return nil
}

func (self *SModelartsPool) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SModelartsPool) IsEmulated() bool {
	return false
}

func (self *SModelartsPool) GetBillingType() string {
	if self.Metadata.Annotations.BillingType == "1" {
		return billing_api.BILLING_TYPE_PREPAID
	} else {
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

// 获取资源归属项目Id
func (self *SModelartsPool) GetProjectId() string {
	return self.Metadata.Name
}

func (self *SModelartsPool) GetExpiredAt() time.Time {
	ret, _ := time.Parse("2006-01-02T15:04:05CST", self.Metadata.CreationTimestamp)
	if !ret.IsZero() {
		ret = ret.Add(time.Hour * 8)
	}
	return ret
}

func (self *SModelartsPool) IsAutoRenew() bool {
	return false
}

func (self *SModelartsPool) Renew(bc billing.SBillingCycle) error {
	return nil
}

func (self *SModelartsPool) SetAutoRenew(bc billing.SBillingCycle) error {
	return nil
}

func (self *SModelartsPool) RefreshForCreate() error {
	self.Status.Resource = SNodeStatus{}
	pool, err := self.region.list(SERVICE_MODELARTS, "pools/"+self.GetId(), nil)
	if err == nil {
		return pool.Unmarshal(self)
	}
	if errors.Cause(err) != errors.ErrNotFound {
		return err
	}

	pools := make([]SModelartsPool, 0)
	params := url.Values{}
	params.Add("status", "failed")
	resObj, err := self.region.list(SERVICE_MODELARTS, "pools", params)
	if err != nil {
		return errors.Wrap(err, "list failed pools")
	}

	err = resObj.Unmarshal(&pools, "items")
	if err != nil {
		return errors.Wrap(err, "resObj unmarshal")
	}

	for _, pool := range pools {
		if pool.GetId() == self.GetId() {
			self.Status.Phase = "CreationFailed"
			return jsonutils.Update(self, pool)
		}
	}
	return err
}

func (self *SModelartsPool) Refresh() error {
	self.Status.Resource = SNodeStatus{}
	resource := fmt.Sprintf("pools/%s", self.GetId())
	pool, err := self.region.list(SERVICE_MODELARTS, resource, nil)
	if err != nil {
		return err
	}
	return pool.Unmarshal(self)
}

func (self *SModelartsPool) SetTags(tags map[string]string, replace bool) error {
	return nil
}

func (self *SModelartsPool) Delete() error {
	_, err := self.region.DeletePool(self.GetId())
	if err != nil {
		return err
	}
	return nil
}

func (self *SModelartsPool) GetInstanceType() string {
	return self.Spec.Resource[0].Flavor
}

func (self *SModelartsPool) GetWorkType() string {
	return strings.Join(self.Spec.Scope, ",")
}

func (self *SModelartsPool) GetNodeCount() int {
	if len(self.Spec.Resource) < 1 {
		return 0
	}
	nodeCount := 0
	for _, v := range self.Spec.Resource {
		nodeCount += v.Count
	}
	return nodeCount
}

func (self *SModelartsPool) ChangeConfig(opts *cloudprovider.ModelartsPoolChangeConfigOptions) error {
	//{"spec":{"resources":[{"flavor":"modelarts.kat1.8xlarge","count":2}]}}
	res := []map[string]interface{}{}
	for _, re := range self.Spec.Resource {
		res = append(res, map[string]interface{}{
			"flavor": re.Flavor,
			"count":  opts.NodeCount,
		})
	}
	params := map[string]interface{}{
		"spec": map[string]interface{}{
			"resources": res,
		},
	}
	resource := fmt.Sprintf("pools/%s", self.Metadata.Name)
	urlValue := url.Values{}
	urlValue.Add("time_range", "")
	urlValue.Add("statistics", "")
	urlValue.Add("period", "")
	_, err := self.region.patch(SERVICE_MODELARTS, resource, urlValue, params)
	return err
}
