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

package azure

import (
	"context"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SLoadbalancerBackend struct {
	multicloud.SResourceBase

	lbbg *SLoadbalancerBackendGroup
	// networkInterfaces 通过接口地址反查虚拟机地址
	Name        string `json:"name"`
	ID          string `json:"id"`
	Type        string `json:"type"`
	BackendPort int
	BackendIP   string
}

func (self *SLoadbalancerBackend) GetId() string {
	return self.ID + "::" + strconv.Itoa(self.BackendPort)
}

func (self *SLoadbalancerBackend) GetName() string {
	return self.Name
}

func (self *SLoadbalancerBackend) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadbalancerBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancerBackend) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadbalancerBackend) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SLoadbalancerBackend) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

func (self *SLoadbalancerBackend) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SLoadbalancerBackend) GetWeight() int {
	return 0
}

func (self *SLoadbalancerBackend) GetPort() int {
	return self.BackendPort
}

func (self *SLoadbalancerBackend) GetBackendType() string {
	return self.Type
}

func (self *SLoadbalancerBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (self *SLoadbalancerBackend) GetBackendId() string {
	return self.ID
}

func (self *SLoadbalancerBackend) GetIpAddress() string {
	return self.BackendIP
}

func (self *SLoadbalancerBackend) SyncConf(ctx context.Context, port, weight int) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SyncConf")
}
