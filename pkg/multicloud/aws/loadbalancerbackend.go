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
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SElbBackend struct {
	region *SRegion
	group  *SElbBackendGroup

	Target       Target       `json:"Target"`
	TargetHealth TargetHealth `json:"TargetHealth"`
}

type Target struct {
	ID   string `json:"Id"`
	Port int    `json:"Port"`
}

type TargetHealth struct {
	State       string `json:"State"`
	Reason      string `json:"Reason"`
	Description string `json:"Description"`
}

func (self *SElbBackend) GetId() string {
	return fmt.Sprintf("%s::%s::%d", self.group.GetId(), self.Target.ID, self.Target.Port)
}

func (self *SElbBackend) GetName() string {
	return self.GetId()
}

func (self *SElbBackend) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbBackend) Refresh() error {
	return nil
}

func (self *SElbBackend) IsEmulated() bool {
	return false
}

func (self *SElbBackend) GetMetadata() *jsonutils.JSONDict {
	return jsonutils.NewDict()
}

func (self *SElbBackend) GetProjectId() string {
	return ""
}

func (self *SElbBackend) GetWeight() int {
	return 0
}

func (self *SElbBackend) GetPort() int {
	return self.Target.Port
}

func (self *SElbBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (self *SElbBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (self *SElbBackend) GetBackendId() string {
	return self.Target.ID
}

func (self *SElbBackend) SyncConf(port, weight int) error {
	return self.region.SyncElbBackend(self.GetId(), self.GetBackendId(), self.Target.Port, port)
}

func (self *SRegion) SyncElbBackend(backendId, serverId string, oldPort, newPort int) error {
	err := self.RemoveElbBackend(backendId, serverId, 0, oldPort)
	if err != nil {
		return err
	}

	_, err = self.AddElbBackend(backendId, serverId, 0, newPort)
	if err != nil {
		return err
	}

	return nil
}
