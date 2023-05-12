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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKubeCluster struct {
	multicloud.SResourceBase
	QcloudTags
	region *SRegion

	ClusterId              string
	ClusterName            string
	ClusterDescription     string
	ClusterVersion         string
	ClusterOs              string
	ClusterType            string
	ClusterNetworkSettings struct {
		VpcId   string
		Subnets []string
	}
	ClusterNodeNum   int
	ProjectId        string
	TagSpecification []struct {
	}
	ClusterStatus       string
	Property            string
	ClusterMaterNodeNum int
	ImageId             string
	OsCustomizeType     string
	ContainerRuntime    string
	CreatedTime         string
	DeletionProtection  bool
	EnableExternalNode  bool
}

func (self *SKubeCluster) GetId() string {
	return self.ClusterId
}

func (self *SKubeCluster) GetGlobalId() string {
	return self.ClusterId
}

func (self *SKubeCluster) GetName() string {
	return self.ClusterName
}

func (self *SKubeCluster) GetStatus() string {
	return strings.ToLower(self.ClusterStatus)
}

func (self *SKubeCluster) GetEnabled() bool {
	return true
}

func (self *SKubeCluster) Refresh() error {
	cluster, err := self.region.GetKubeCluster(self.ClusterId)
	if err != nil {
		return errors.Wrapf(err, "GetKubeCluster(%s)", self.ClusterId)
	}
	return jsonutils.Update(self, cluster)
}

func (self *SKubeCluster) GetVersion() string {
	return self.ClusterVersion
}

func (self *SKubeCluster) GetVpcId() string {
	return self.ClusterNetworkSettings.VpcId
}

func (self *SKubeCluster) GetNetworkIds() []string {
	return self.ClusterNetworkSettings.Subnets
}

func (self *SKubeCluster) GetKubeConfig(private bool, expireMinutes int) (*cloudprovider.SKubeconfig, error) {
	return self.region.GetKubeConfig(self.ClusterId, private)
}

func (self *SKubeCluster) Delete(isRetain bool) error {
	return self.region.DeleteKubeCluster(self.ClusterId, isRetain)
}

func (self *SRegion) GetKubeConfig(clusterId string, private bool) (*cloudprovider.SKubeconfig, error) {
	params := map[string]string{
		"ClusterId":  clusterId,
		"IsExtranet": fmt.Sprintf("%v", !private),
	}
	resp, err := self.tkeRequest("DescribeClusterKubeconfig", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeClusterKubeconfig")
	}
	config, err := resp.GetString("Kubeconfig")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.GetKubeconfig")
	}
	result := &cloudprovider.SKubeconfig{}
	result.Config = config
	return result, nil
}

func (self *SRegion) GetICloudKubeClusters() ([]cloudprovider.ICloudKubeCluster, error) {
	clusters := []SKubeCluster{}
	for {
		part, total, err := self.GetKubeClusters(nil, 100, len(clusters)/100)
		if err != nil {
			return nil, errors.Wrapf(err, "GetKubeClusters")
		}
		clusters = append(clusters, part...)
		if len(clusters) >= total || len(part) == 0 {
			break
		}
	}
	ret := []cloudprovider.ICloudKubeCluster{}
	for i := range clusters {
		clusters[i].region = self
		ret = append(ret, &clusters[i])
	}
	return ret, nil
}

func (self *SRegion) GetKubeClusters(ids []string, limit, offset int) ([]SKubeCluster, int, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	params := map[string]string{
		"Limit":  fmt.Sprintf("%d", limit),
		"Offset": fmt.Sprintf("%d", offset),
	}
	for i, id := range ids {
		params[fmt.Sprintf("ClusterIds.%d", i)] = id
	}
	resp, err := self.tkeRequest("DescribeClusters", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeClusters")
	}
	clusters := []SKubeCluster{}
	err = resp.Unmarshal(&clusters, "Clusters")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Float("TotalCount")
	return clusters, int(totalCount), nil
}

func (self *SRegion) GetICloudKubeClusterById(id string) (cloudprovider.ICloudKubeCluster, error) {
	cluster, err := self.GetKubeCluster(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetKubeCluster(%s)", id)
	}
	return cluster, nil
}

func (self *SRegion) GetKubeCluster(id string) (*SKubeCluster, error) {
	clusters, total, err := self.GetKubeClusters([]string{id}, 1, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "GetKubeCluster(%s)", id)
	}
	if total == 1 {
		clusters[0].region = self
		return &clusters[0], nil
	}
	if total == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}
	return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, id)
}

func (self *SRegion) DeleteKubeCluster(id string, isRetain bool) error {
	params := map[string]string{
		"ClusterId":          id,
		"InstanceDeleteMode": "retain",
	}
	if !isRetain {
		params["InstanceDeleteMode"] = "terminate"
		params["ResourceDeleteOptions.0.ResourceType"] = "CBS"
		params["ResourceDeleteOptions.0.DeleteMode"] = "terminate"
	}
	_, err := self.tkeRequest("DeleteCluster", params)
	return errors.Wrapf(err, "DeleteCluster")
}

func (self *SKubeCluster) CreateIKubeNodePool(opts *cloudprovider.KubeNodePoolCreateOptions) (cloudprovider.ICloudKubeNodePool, error) {
	return nil, cloudprovider.ErrNotImplemented
}
