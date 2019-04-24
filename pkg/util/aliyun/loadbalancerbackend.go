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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerBackend struct {
	lbbg *SLoadbalancerBackendGroup

	ServerId string
	Port     int
	Weight   int
}

func (backend *SLoadbalancerBackend) GetName() string {
	return backend.ServerId
}

func (backend *SLoadbalancerBackend) GetId() string {
	return fmt.Sprintf("%s/%s", backend.lbbg.VServerGroupId, backend.ServerId)
}

func (backend *SLoadbalancerBackend) GetGlobalId() string {
	return backend.GetId()
}

func (backend *SLoadbalancerBackend) GetStatus() string {
	return ""
}

func (backend *SLoadbalancerBackend) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backend *SLoadbalancerBackend) IsEmulated() bool {
	return false
}

func (backend *SLoadbalancerBackend) Refresh() error {
	loadbalancerBackends, err := backend.lbbg.lb.region.GetLoadbalancerBackends(backend.lbbg.VServerGroupId)
	if err != nil {
		return err
	}
	for _, loadbalancerBackend := range loadbalancerBackends {
		if loadbalancerBackend.ServerId == backend.ServerId {
			return jsonutils.Update(backend, loadbalancerBackend)
		}
	}
	return cloudprovider.ErrNotFound
}

func (backend *SLoadbalancerBackend) GetWeight() int {
	return backend.Weight
}

func (backend *SLoadbalancerBackend) GetPort() int {
	return backend.Port
}

func (backend *SLoadbalancerBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (backend *SLoadbalancerBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (backend *SLoadbalancerBackend) GetBackendId() string {
	return backend.ServerId
}

func (region *SRegion) GetLoadbalancerBackends(backendgroupId string) ([]SLoadbalancerBackend, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["VServerGroupId"] = backendgroupId
	body, err := region.lbRequest("DescribeVServerGroupAttribute", params)
	if err != nil {
		return nil, err
	}
	backends := []SLoadbalancerBackend{}
	return backends, body.Unmarshal(&backends, "BackendServers", "BackendServer")
}

func (backend *SLoadbalancerBackend) GetProjectId() string {
	return ""
}
