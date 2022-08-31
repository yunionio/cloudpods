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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SModelartsPool struct {
	client *SHuaweiClient

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

type SModelartsPoolStatus struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

func (self *SHuaweiClient) GetIModelartsPools() ([]cloudprovider.ICloudModelartsPool, error) {
	pools := make([]SModelartsPool, 0)
	resObj, err := self.modelartsPoolList("pools", nil)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetPools")
	}
	err = resObj.Unmarshal(&pools, "items")
	if err != nil {
		return nil, errors.Wrap(err, "resObj unmarshal")
	}
	res := make([]cloudprovider.ICloudModelartsPool, len(pools))
	for i := 0; i < len(pools); i++ {
		res[i] = &pools[i]
	}
	log.Infoln(res)
	return res, nil
}

func (self *SHuaweiClient) CreateIModelartsPool(args *cloudprovider.ModelartsPoolCreateOption) (cloudprovider.ICloudModelartsPool, error) {
	scopeArr := strings.Split(args.WorkType, ",")
	params := map[string]interface{}{
		"apiVersion": "v2",
		"kind":       "Pool",
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				"os.modelarts/name":         args.Name, //pool name
				"os.modelarts/workspace.id": "0",
			},
		},
		"spec": map[string]interface{}{
			"type":  "Dedicate",
			"scope": scopeArr,
			"network": map[string]interface{}{
				"name": "test-4f954e9555964f019f88813161540828",
			},

			"resources": []map[string]interface{}{
				{
					"flavor": args.InstanceType, // "modelarts.vm.cpu.8ud", // resourceType
					"count":  1,                 //
				},
			},
		},
	}
	obj, err := self.modelartsPoolCreate("pools", params)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetPools")
	}
	pool := &SModelartsPool{}
	obj.Unmarshal(&pool)
	res := []cloudprovider.ICloudModelartsPool{}
	for i := 0; i < 1; i++ {
		pool.client = self
		res = append(res, pool)
	}

	return res[0], nil
}

func (self *SHuaweiClient) DeletePool(poolName string) (jsonutils.JSONObject, error) {
	return self.modelartsPoolDelete("pools", poolName, nil)
}

func (self *SHuaweiClient) Update(args *cloudprovider.ModelartsPoolUpdateOption) (cloudprovider.ICloudModelartsPool, error) {
	scopeArr := strings.Split(args.WorkType, ",")
	params := map[string]interface{}{
		"spec": map[string]interface{}{
			"scope": scopeArr,
		},
	}
	obj, err := self.modelartsPoolUpdate(args.Id, params)
	if err != nil {
		return nil, errors.Wrap(err, "modelartsPoolUpdate")
	}
	pool := &SModelartsPool{}
	obj.Unmarshal(&pool)
	res := []cloudprovider.ICloudModelartsPool{}
	for i := 0; i < 1; i++ {
		pool.client = self
		res = append(res, pool)
	}
	return res[0], nil
}

func (self *SHuaweiClient) GetIModelartsPoolById(poolId string) (cloudprovider.ICloudModelartsPool, error) {
	obj, err := self.modelartsPoolById(poolId, nil)
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
		pool.client = self
		res = append(res, pool)
	}
	return res[0], nil
}

func (self *SHuaweiClient) MonitorPool(poolId string) (*SModelartsMetrics, error) {
	resObj, err := self.modelartsPoolMonitor(poolId, nil)
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

func (self *SHuaweiClient) CreatePoolNetworks(poolName string) (jsonutils.JSONObject, error) {
	params := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Network",
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				"os.modelarts/name": "test",
			},
		},
		"spec": map[string]interface{}{
			"cidr": "192.168.0.0/16",
			"connection": map[string]interface{}{
				"peerConnectionList": []map[string]interface{}{
					{"peerVpcId": "c502dd2c-c6e6-4c50-9f5b-81e29a76fb29",
						"peerSubnetid": "774ebb82-b26a-4fe5-b3ae-e65727f67d36"},
				},
			},
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
	return strings.ToLower(self.Status.Phase)
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
	return self.Metadata.Annotations.BillingType
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
	pool, err := self.client.modelartsPoolById(self.GetId(), nil)
	if err != nil {
		return errors.Wrapf(err, "GetModelartsPool(%s)", self.GetId())
	}
	return jsonutils.Update(self, pool)
}

func (self *SModelartsPool) SetTags(tags map[string]string, replace bool) error {
	// return self.client.SetResourceTags(ALIYUN_SERVICE_NAS, "filesystem", self.FileSystemId, tags, replace)
	return nil
}

func (self *SModelartsPool) Delete() error {
	_, err := self.client.DeletePool(self.GetId())
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
