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
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancerDefaultBackend struct {
	multicloud.SResourceBase
	AliyunTags
	lbbg *SLoadbalancerDefaultBackendGroup

	ServerId string
	Weight   int
}

func (backend *SLoadbalancerDefaultBackend) GetName() string {
	return backend.ServerId
}

func (backend *SLoadbalancerDefaultBackend) GetId() string {
	return fmt.Sprintf("%s/%s", backend.lbbg.lb.LoadBalancerId, backend.ServerId)
}

func (backend *SLoadbalancerDefaultBackend) GetGlobalId() string {
	return backend.GetId()
}

func (backend *SLoadbalancerDefaultBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (backend *SLoadbalancerDefaultBackend) IsEmulated() bool {
	return false
}

func (backend *SLoadbalancerDefaultBackend) Refresh() error {
	return nil
}

func (backend *SLoadbalancerDefaultBackend) GetWeight() int {
	return backend.Weight
}

func (backend *SLoadbalancerDefaultBackend) GetPort() int {
	return 0
}

func (backend *SLoadbalancerDefaultBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (backend *SLoadbalancerDefaultBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (backend *SLoadbalancerDefaultBackend) GetBackendId() string {
	return backend.ServerId
}

func (backend *SLoadbalancerDefaultBackend) GetIpAddress() string {
	return ""
}

func (backend *SLoadbalancerDefaultBackend) GetProjectId() string {
	return backend.lbbg.GetProjectId()
}

func (backend *SLoadbalancerDefaultBackend) SyncConf(ctx context.Context, port, weight int) error {
	params := map[string]string{}
	params["RegionId"] = backend.lbbg.lb.region.RegionId
	params["LoadBalancerId"] = backend.lbbg.lb.LoadBalancerId
	loadbalancer, err := backend.lbbg.lb.region.GetLoadbalancerDetail(backend.lbbg.lb.LoadBalancerId)
	if err != nil {
		return err
	}
	servers := jsonutils.NewArray()
	for i := 0; i < len(loadbalancer.BackendServers.BackendServer); i++ {
		_backend := loadbalancer.BackendServers.BackendServer[i]
		_backend.lbbg = backend.lbbg
		if _backend.GetGlobalId() == backend.GetGlobalId() {
			_backend.Weight = weight
		}
		servers.Add(
			jsonutils.Marshal(
				map[string]string{
					"ServerId": _backend.ServerId,
					"Weight":   fmt.Sprintf("%d", _backend.Weight),
				},
			))
	}

	params["BackendServers"] = servers.String()
	_, err = backend.lbbg.lb.region.lbRequest("SetBackendServers", params)
	return err
}
