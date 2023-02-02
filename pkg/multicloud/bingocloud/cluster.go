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

package bingocloud

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SCluster struct {
	multicloud.SResourceBase
	multicloud.STagBase

	region *SRegion

	ClusterControllerSet []struct {
		Address string
		Role    string
	}
	ClusterId        string
	ClusterName      string
	CreateVolumeMode string
	ExtendDiskMode   string
	Hypervisor       string
	MaxVolumeStorage string
	SchedPolicy      string
	Status           string
}

func (self *SCluster) GetGlobalId() string {
	return self.ClusterId
}

func (self *SCluster) GetId() string {
	return self.ClusterId
}

func (self *SCluster) GetName() string {
	return self.ClusterName
}

func (self *SCluster) GetStatus() string {
	if self.Status == "available" {
		return api.ZONE_ENABLE
	}
	return api.ZONE_DISABLE
}

func (self *SCluster) GetI18n() cloudprovider.SModelI18nTable {
	return cloudprovider.SModelI18nTable{}
}

func (self *SCluster) GetIRegion() cloudprovider.ICloudRegion {
	return self.region
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	clusters, err := self.GetClusters()
	if err != nil {
		return nil, errors.Wrapf(err, "GetClusters")
	}
	var ret []cloudprovider.ICloudZone
	for i := range clusters {
		clusters[i].region = self
		ret = append(ret, &clusters[i])
	}
	return ret, nil
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	zones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].GetGlobalId() == id {
			return zones[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetClusters() ([]SCluster, error) {
	resp, err := self.invoke("DescribeClusters", nil)
	if err != nil {
		return nil, err
	}
	var ret []SCluster
	return ret, resp.Unmarshal(&ret, "clusterSet")
}
