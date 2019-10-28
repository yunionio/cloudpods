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
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerMasterSlaveBackend struct {
	lbbg *SLoadbalancerMasterSlaveBackendGroup

	ServerId   string
	Weight     int
	Port       int
	ServerType string
}

func (backend *SLoadbalancerMasterSlaveBackend) GetName() string {
	return backend.ServerId
}

func (backend *SLoadbalancerMasterSlaveBackend) GetId() string {
	return fmt.Sprintf("%s/%s", backend.lbbg.MasterSlaveServerGroupId, backend.ServerId)
}

func (backend *SLoadbalancerMasterSlaveBackend) GetGlobalId() string {
	return backend.GetId()
}

func (backend *SLoadbalancerMasterSlaveBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (backend *SLoadbalancerMasterSlaveBackend) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backend *SLoadbalancerMasterSlaveBackend) IsEmulated() bool {
	return false
}

func (backend *SLoadbalancerMasterSlaveBackend) Refresh() error {
	return nil
}

func (backend *SLoadbalancerMasterSlaveBackend) GetWeight() int {
	return backend.Weight
}

func (backend *SLoadbalancerMasterSlaveBackend) GetPort() int {
	return backend.Port
}

func (backend *SLoadbalancerMasterSlaveBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (backend *SLoadbalancerMasterSlaveBackend) GetBackendRole() string {
	return strings.ToLower(backend.ServerType)
}

func (backend *SLoadbalancerMasterSlaveBackend) GetBackendId() string {
	return backend.ServerId
}

func (backend *SLoadbalancerMasterSlaveBackend) GetProjectId() string {
	return ""
}

func (backend *SLoadbalancerMasterSlaveBackend) SyncConf(port, weight int) error {
	return cloudprovider.ErrNotSupported
}
