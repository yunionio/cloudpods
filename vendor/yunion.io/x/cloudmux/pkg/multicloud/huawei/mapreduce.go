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
)

type SMapReduceCluster struct {
	ClusterId   string
	ClusterName string
}

func (self *SRegion) ListMapReduceCluster() ([]SMapReduceCluster, error) {
	query := url.Values{}
	query.Set("pageSize", "200")
	ret := []SMapReduceCluster{}
	currentPage := 1
	for {
		query.Set("currentPage", fmt.Sprintf("%d", currentPage))
		resp, err := self.list(SERVICE_MRS, "cluster_infos", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Clusters     []SMapReduceCluster
			ClusterTotal int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Clusters...)
		if len(ret) >= part.ClusterTotal || len(part.Clusters) == 0 {
			break
		}
		currentPage++
	}
	return ret, nil
}
