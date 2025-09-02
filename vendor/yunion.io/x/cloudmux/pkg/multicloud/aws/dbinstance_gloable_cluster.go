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

package aws

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SDBInstanceGlobalCluster struct {
	multicloud.SDBInstanceBase
	AwsTags

	region *SRegion

	GlobalClusterIdentifier string `xml:"GlobalClusterIdentifier"`
}

func (region *SRegion) GetDBInstanceGlobalClusters(id string) ([]SDBInstanceGlobalCluster, error) {
	ret := []SDBInstanceGlobalCluster{}
	params := map[string]string{}
	if len(id) > 0 {
		params["GlobalClusterIdentifier"] = id
	}
	for {
		part := struct {
			GlobalClusters []SDBInstanceGlobalCluster `xml:"GlobalClusters>GlobalCluster"`
			Marker         string                     `xml:"Marker"`
		}{}
		err := region.rdsRequest("DescribeGlobalClusters", params, &part)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeGlobalClusters")
		}
		ret = append(ret, part.GlobalClusters...)
		if len(part.GlobalClusters) == 0 || len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}

	return ret, nil
}

func (region *SRegion) GetDBInstanceGlobalCluster(id string) (*SDBInstanceGlobalCluster, error) {
	clusters, err := region.GetDBInstanceGlobalClusters(id)
	if err != nil {
		return nil, err
	}

	for i := range clusters {
		if clusters[i].GlobalClusterIdentifier == id {
			clusters[i].region = region
			return &clusters[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}
