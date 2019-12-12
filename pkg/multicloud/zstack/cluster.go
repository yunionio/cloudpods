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

package zstack

import "net/url"

type SCluster struct {
	ZStackBasic
	Description    string `json:"description"`
	State          string `json:"State"`
	HypervisorType string `json:"hypervisorType"`
	ZStackTime
	ZoneUUID string `json:"zoneUuid"`
	Type     string `json:"type"`
}

func (region *SRegion) GetClusters() ([]SCluster, error) {
	clusters := []SCluster{}
	params := url.Values{}
	if SkipEsxi {
		params.Set("q", "type!=vmware")
	}
	return clusters, region.client.listAll("clusters", params, &clusters)
}

func (region *SRegion) GetClusterIds() ([]string, error) {
	clusters, err := region.GetClusters()
	if err != nil {
		return nil, err
	}
	clusterIds := []string{}
	for i := 0; i < len(clusters); i++ {
		clusterIds = append(clusterIds, clusters[i].UUID)
	}
	return clusterIds, nil
}
