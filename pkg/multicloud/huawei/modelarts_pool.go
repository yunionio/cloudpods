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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// 算力资源池
type SModelartsPool struct {
	client *SHuaweiClient

	Metadata SModelartsPoolMetadata `json:"metadata"`
	Spec     SModelartsPoolSpec     `json:"spec"`
	Status   SModelartsPoolStatus   `json:"status"`
}

type SModelartsPoolMetadata struct {
	Name              string `json:"name"`
	CreationTimestamp string `json:"creationTimestamp"`
	Labels            SModelartsPoolMeatadataLabel
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
	Resource []SModelartsPoolResource `json:"resource"`
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

type PredefinedFlavors struct {
}

type PoolSfsTurbo struct {
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
	return res, nil
}

func (self *SHuaweiClient) CreateIModelartsPool(args *cloudprovider.ModelartsPoolCreateOption) (cloudprovider.ICloudModelartsPool, error) {
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
			"scope": []string{"Train"},
			"network": map[string]interface{}{
				"name": "test-4f954e9555964f019f88813161540828",
			},

			"resources": []map[string]interface{}{
				{
					"flavor": args.ResourceFlavor, // "modelarts.vm.cpu.8ud", // resourceType
					"count":  args.ResourceCount,  //
				},
			},
		},
	}
	res, err := self.modelartsPoolCreate("pools", params)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetPools")
	}
	id, err := res.GetString("metadata", "name")
	if err != nil {
		return nil, errors.Wrap(err, "metadata.name")
	}
	pool, err := self.GetIModelartsPoolDetail(id)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetIModelartsPoolDetail")
	}
	return pool, nil
}

func (self *SHuaweiClient) DeletePool(poolName string) (jsonutils.JSONObject, error) {
	return self.modelartsPoolDelete("pools", poolName, nil)
}

func (self *SHuaweiClient) UpdatePool(poolName string) (jsonutils.JSONObject, error) {
	return self.modelartsPoolUpdate(poolName, nil)
}

func (self *SHuaweiClient) GetIModelartsPoolDetail(poolId string) (cloudprovider.ICloudModelartsPool, error) {
	obj, err := self.modelartsPoolById(poolId, nil)
	if err != nil {
		return nil, errors.Wrap(err, "region.modelartsPoolByName")
	}
	pool := &SModelartsPool{}
	obj.Unmarshal(&pool)
	res := []cloudprovider.ICloudModelartsPool{}
	for i := 0; i < 1; i++ {
		res[i] = pool
	}
	return res[0], nil
}

type SModelArtsMetrics struct {
	Metrics []SModelArtsMetric `json:"metrics"`
}

type SModelArtsMetric struct {
	SModelArtsMetricInfo `json:"metric"`
	SModelArtsDataPoints `json:"dataPoints"`
}

type SModelArtsMetricInfo struct {
	Dimensions SModelArtsDimensions `json:"dimensions"`
	MetricName string
	Namespace  string
}

type SModelArtsDimensions struct {
	Name  string
	Value string
}

type SModelArtsDataPoints struct {
	Timestamp  uint64
	Unit       string
	Statistics []ModelArtsStatistics
}

type ModelArtsStatistics struct {
	Statistic string
	Value     int
}

func (self *SRegion) MonitorPool(poolId string) (*SModelArtsMetrics, error) {
	resObj, err := self.client.modelartsPoolMonitor(poolId, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "send request error")
	}
	metrics := SModelArtsMetrics{}
	err = resObj.Unmarshal(&metrics)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarsh error")
	}
	return &metrics, nil
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
	return self.Status.Phase
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

func (self *SModelartsPool) Refresh() error {
	fs, err := self.client.modelartsPoolById(self.GetId(), nil)
	if err != nil {
		return errors.Wrapf(err, "GetFileSystem(%s)", self.GetId())
	}
	return jsonutils.Update(self, fs)
}

func (self *SModelartsPool) SetTags(tags map[string]string, replace bool) error {
	// return self.client.SetResourceTags(ALIYUN_SERVICE_NAS, "filesystem", self.FileSystemId, tags, replace)
	return nil
}
