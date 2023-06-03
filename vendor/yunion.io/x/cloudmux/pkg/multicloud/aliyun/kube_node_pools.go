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

package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKubeNodePool struct {
	multicloud.SResourceBase
	AliyunTags

	cluster *SKubeCluster

	AutoScaling struct {
		EipBandwidth int    `json:"eip_bandwidth"`
		IsBondEip    bool   `json:"is_bond_eip"`
		Enable       bool   `json:"enable"`
		MaxInstances int    `json:"max_instances"`
		MinInstances int    `json:"min_instances"`
		Type         string `json:"type"`
	} `json:"auto_scaling"`
	KubernetesConfig struct {
		CmsEnabled     bool   `json:"cms_enabled"`
		CpuPolicy      string `json:"cpu_policy"`
		Runtime        string `json:"runtime"`
		RuntimeVersion string `json:"runtime_version"`
		UserData       string `json:"user_data"`
	} `json:"kubernetes_config"`
	NodepoolInfo struct {
		Created         string `json:"created"`
		IsDefault       bool   `json:"is_default"`
		Name            string `json:"name"`
		NodepoolId      string `json:"nodepool_id"`
		RegionId        string `json:"region_id"`
		ResourceGroupId string `json:"resource_group_id"`
		Type            string `json:"type"`
		Updated         string `json:"updated"`
	} `json:"nodepool_info"`
	ScalingGroup struct {
		AutoRenew                           bool     `json:"auto_renew"`
		AutoRenewPeriod                     int      `json:"auto_renew_period"`
		ImageId                             string   `json:"image_id"`
		InstanceChargeType                  string   `json:"instance_charge_type"`
		InstanceTypes                       []string `json:"instance_types"`
		MultiAzPolicy                       string   `json:"multi_az_policy"`
		OnDemandBaseCapacity                int      `json:"on_demand_base_capacity"`
		OnDemandPercentageAboveBaseCapacity int      `json:"on_demand_percentage_above_base_capacity"`
		SpotInstancePools                   int      `json:"spot_instance_pools"`
		SpotInstanceRemedy                  bool     `json:"spot_instance_remedy"`
		CompensateWithOnDemand              bool     `json:"compensate_with_on_demand"`
		Period                              int      `json:"period"`
		PeriodUnit                          string   `json:"period_unit"`
		Platform                            string   `json:"platform"`
		RamPolicy                           string   `json:"ram_policy"`
		SpotStrategy                        string   `json:"spot_strategy"`
		SpotPriceLimit                      []struct {
			InstanceType string `json:"instance_type"`
			PriceLimit   string `json:"price_limit"`
		} `json:"spot_price_limit"`
		RdsInstances       []string `json:"rds_instances"`
		ScalingGroupId     string   `json:"scaling_group_id"`
		ScalingPolicy      string   `json:"scaling_policy"`
		SecurityGroupId    string   `json:"security_group_id"`
		SystemDiskCategory string   `json:"system_disk_category"`
		SystemDiskSize     int      `json:"system_disk_size"`
		VswitchIds         []string `json:"vswitch_ids"`
		LoginPassword      string   `json:"login_password"`
		KeyPair            string   `json:"key_pair"`
		DesiredSize        int      `json:"desired_size"`
	} `json:"scaling_group"`
	Status struct {
		FailedNodes   string `json:"failed_nodes"`
		HealthyNodes  string `json:"healthy_nodes"`
		InitialNodes  string `json:"initial_nodes"`
		OfflineNodes  string `json:"offline_nodes"`
		RemovingNodes string `json:"removing_nodes"`
		ServingNodes  string `json:"serving_nodes"`
		State         string `json:"state"`
		TotalNodes    int    `json:"total_nodes"`
	} `json:"status"`
	TeeConfig struct {
		TeeEnable bool `json:"tee_enable"`
	} `json:"tee_config"`
	Management struct {
		Enable        bool `json:"enable"`
		AutoRepair    bool `json:"auto_repair"`
		UpgradeConfig struct {
			AutoUpgrade     bool `json:"auto_upgrade"`
			Surge           int  `json:"surge"`
			SurgePercentage int  `json:"surge_percentage"`
			MaxUnavailable  int  `json:"max_unavailable"`
		} `json:"upgrade_config"`
	} `json:"management"`
}

func (self *SKubeNodePool) GetName() string {
	return self.NodepoolInfo.Name
}

func (self *SKubeNodePool) GetId() string {
	return self.NodepoolInfo.NodepoolId
}

func (self *SKubeNodePool) GetGlobalId() string {
	return self.NodepoolInfo.NodepoolId
}

func (self *SKubeNodePool) Refresh() error {
	pool, err := self.cluster.region.GetKubeNodePool(self.cluster.ClusterId, self.NodepoolInfo.NodepoolId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, pool)
}

func (self *SKubeNodePool) GetStatus() string {
	switch self.Status.State {
	case "active":
		return api.KUBE_CLUSTER_STATUS_RUNNING
	}
	return self.Status.State
}

func (self *SKubeNodePool) GetMinInstanceCount() int {
	return self.AutoScaling.MinInstances
}

func (self *SKubeNodePool) GetMaxInstanceCount() int {
	return self.AutoScaling.MaxInstances
}

func (self *SKubeNodePool) GetDesiredInstanceCount() int {
	return self.ScalingGroup.DesiredSize
}

func (self *SKubeNodePool) GetRootDiskSizeGb() int {
	return self.ScalingGroup.SystemDiskSize
}

func (self *SKubeNodePool) Delete() error {
	return self.cluster.region.DeleteKubeNodePool(self.cluster.ClusterId, self.NodepoolInfo.NodepoolId)
}

func (self *SKubeNodePool) GetInstanceTypes() []string {
	return self.ScalingGroup.InstanceTypes
}

func (self *SKubeNodePool) GetNetworkIds() []string {
	return self.ScalingGroup.VswitchIds
}

func (self *SRegion) DeleteKubeNodePool(clusterId, id string) error {
	params := map[string]string{
		"PathPattern": fmt.Sprintf("/clusters/%s/nodepools/%s", clusterId, id),
	}
	_, err := self.k8sRequest("DeleteClusterNodepool", params, map[string]string{})
	return errors.Wrapf(err, "DeleteCluster")
}

func (self *SRegion) GetKubeNodePool(clusterId, id string) (*SKubeNodePool, error) {
	params := map[string]string{
		"PathPattern": fmt.Sprintf("/clusters/%s/nodepools/%s", clusterId, id),
	}
	resp, err := self.k8sRequest("DescribeClusterNodePoolDetail", params, map[string]string{})
	if err != nil {
		return nil, err
	}
	ret := &SKubeNodePool{}
	return ret, resp.Unmarshal(ret)
}

func (self *SKubeCluster) GetIKubeNodePools() ([]cloudprovider.ICloudKubeNodePool, error) {
	pools, err := self.region.GetKubeNodePools(self.ClusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetKubeNodePools")
	}
	ret := []cloudprovider.ICloudKubeNodePool{}
	for i := range pools {
		pools[i].cluster = self
		ret = append(ret, &pools[i])
	}
	return ret, nil
}

func (self *SRegion) GetKubeNodePools(clusterId string) ([]SKubeNodePool, error) {
	params := map[string]string{
		"PathPattern": fmt.Sprintf("/clusters/%s/nodepools", clusterId),
	}
	resp, err := self.k8sRequest("DescribeClusterNodePools", params, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeClusterNodePools")
	}
	pools := []SKubeNodePool{}
	err = resp.Unmarshal(&pools, "nodepools")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return pools, nil
}
