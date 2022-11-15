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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
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
}

type SModelartsPoolNetworkMetadata struct {
	Name              string `json:"name"`
	CreationTimestamp string `json:"creationTimestamp"`
}

func (self *SRegion) GetIModelartsPools() ([]cloudprovider.ICloudModelartsPool, error) {
	pools := make([]SModelartsPool, 0)
	resObj, err := self.client.modelartsPoolList("pools", nil)
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

func (self *SRegion) CreateIModelartsPool(args *cloudprovider.ModelartsPoolCreateOption) (cloudprovider.ICloudModelartsPool, error) {
	netObj, err := self.client.modelartsPoolNetworkList("network", nil)
	if err != nil {
		return nil, errors.Wrap(err, "SHuaweiClient.GetPools")
	}
	netRes := make([]SModelartsPoolNetwork, 0)
	netObj.Unmarshal(&netRes, "items")
	netId := ""
	if len(netRes) != 0 {
		netId = netRes[0].Metadata.Name
	} else {
		createNetObj, err := self.client.CreatePoolNetworks()
		if err != nil {
			return nil, errors.Wrap(err, "SHuaweiClient.CreatePoolNetworks")
		}
		netId, _ = createNetObj.GetString("metadata", "name")
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
	obj, err := self.client.modelartsPoolCreate("pools", params)
	if err != nil {
		return nil, errors.Wrap(err, "SHuaweiClient.CreatePools")
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

func (self *SRegion) DeletePool(poolName string) (jsonutils.JSONObject, error) {
	return self.client.modelartsPoolDelete("pools", poolName, nil)
}

func (self *SRegion) GetIModelartsPoolById(poolId string) (cloudprovider.ICloudModelartsPool, error) {
	obj, err := self.client.modelartsPoolById(poolId)
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
	resObj, err := self.client.modelartsPoolMonitor(poolId, nil)
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
	return self.modelartsPoolNetworkList(poolName, nil)
}

func (self *SHuaweiClient) CreatePoolNetworks() (jsonutils.JSONObject, error) {
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
			// "cidr": "192.168.20.0/24",
			"cidr": "192.168.128.0/17",
		},
	}
	return self.modelartsPoolNetworkCreate(params)
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
	switch {
	case res == compute.MODELARTS_POOL_STATUS_RUNNING && len(self.Status.Resource.Creating) != 0:
		res = compute.MODELARTS_POOL_STATUS_CREATING
	case self.Status.Phase == "CreationFailed":
		res = compute.MODELARTS_POOL_STATUS_CREATE_FAILED
	case self.Status.Phase == "SeclingFailed":
		res = compute.MODELARTS_POOL_STATUS_CHANGE_CONFIG_FAILED
	}
	return res
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

func (self *SModelartsPool) Refresh() error {
	pools := make([]SModelartsPool, 0)
	resObj, err := self.region.client.modelartsPoolListWithStatus("pools", "failed", nil)
	if err != nil {
		return errors.Wrap(err, "modelartsPoolListWithStatus")
	}
	err = resObj.Unmarshal(&pools, "items")
	if err != nil {
		return errors.Wrap(err, "resObj unmarshal")
	}
	for _, pool := range pools {
		if pool.GetId() == self.GetId() {
			self.Status.Phase = "CreationFailed"
		}
	}
	self.Status.Resource = SNodeStatus{}
	pool, err := self.region.client.modelartsPoolById(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "GetModelartsPool(%s)", self.GetId())
	}
	return jsonutils.Update(self, pool)
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
	return self.Spec.Resource[0].Count
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
	_, err := self.region.client.modelartsPoolUpdate(self.Metadata.Name, params)
	return err
}
