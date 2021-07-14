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
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// todo: 虚拟机规模集不支持
// 注: 因为与onecloud后端服务器组存在配置差异，不支持同步未关联的后端服务器组
// 应用型LB：  HTTP 设置 + 后端池 = onecloud 后端服务器组
// 4层LB: loadBalancingRules(backendPort)+ 后端池 = onecloud 后端服务器组
type SLoadbalancerBackendGroup struct {
	lb   *SLoadbalancer
	lbbs []cloudprovider.ICloudLoadbalancerBackend

	Pool         BackendAddressPool
	DefaultPort  int
	HttpSettings *BackendHTTPSettingsCollection

	BackendIps []BackendIPConfiguration
}

func (self *SLoadbalancerBackendGroup) GetId() string {
	return self.Pool.ID + "::" + strconv.Itoa(self.DefaultPort)
}

func (self *SLoadbalancerBackendGroup) GetName() string {
	if self.HttpSettings != nil {
		return self.Pool.Name + "::" + self.HttpSettings.Name
	}

	return self.Pool.Name + "::" + strconv.Itoa(self.DefaultPort)
}

func (self *SLoadbalancerBackendGroup) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadbalancerBackendGroup) GetStatus() string {
	switch self.Pool.Properties.ProvisioningState {
	case "Succeeded":
		return api.LB_STATUS_ENABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadbalancerBackendGroup) Refresh() error {
	lbbg, err := self.lb.GetILoadBalancerBackendGroupById(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetILoadBalancerBackendGroupById")
	}

	err = jsonutils.Update(self, lbbg)
	if err != nil {
		return errors.Wrap(err, "refresh.Update")
	}

	self.lbbs = nil
	return nil
}

func (self *SLoadbalancerBackendGroup) IsEmulated() bool {
	return true
}

func (self *SLoadbalancerBackendGroup) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadbalancerBackendGroup) GetTags() (map[string]string, error) {
	return map[string]string{}, nil
}

func (self *SLoadbalancerBackendGroup) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

func (self *SLoadbalancerBackendGroup) GetProjectId() string {
	return getResourceGroup(self.GetId())
}

func (self *SLoadbalancerBackendGroup) IsDefault() bool {
	return false
}

func (self *SLoadbalancerBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (self *SLoadbalancerBackendGroup) GetLoadbalancerId() string {
	return self.lb.GetId()
}

func (self *SLoadbalancerBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	if self.lbbs != nil {
		return self.lbbs, nil
	}

	var ret []cloudprovider.ICloudLoadbalancerBackend
	ips := self.Pool.Properties.BackendIPConfigurations
	for i := range ips {
		ip := ips[i]
		nic, err := self.lb.region.GetNetworkInterface(strings.Split(ip.ID, "/ipConfigurations")[0])
		if err != nil {
			return nil, errors.Wrap(err, "GetNetworkInterface")
		}

		if len(nic.Properties.VirtualMachine.ID) == 0 {
			continue
		}

		name := nic.Properties.VirtualMachine.Name
		vid := nic.Properties.VirtualMachine.ID
		if len(name) == 0 && len(vid) > 0 {
			segs := strings.Split(vid, "/virtualMachines/")
			name = segs[len(segs)-1]
		}
		bg := SLoadbalancerBackend{
			SResourceBase: multicloud.SResourceBase{},
			lbbg:          self,
			Name:          name,
			ID:            vid,
			Type:          api.LB_BACKEND_GUEST,
			BackendPort:   self.DefaultPort,
		}

		ret = append(ret, &bg)
	}

	ips2 := self.Pool.Properties.BackendAddresses
	for i := range ips2 {
		name := fmt.Sprintf("ip-%s", ips2[i].IPAddress)
		bg := SLoadbalancerBackend{
			SResourceBase: multicloud.SResourceBase{},
			lbbg:          self,
			Name:          name,
			ID:            fmt.Sprintf("%s-%s", self.GetId(), name),
			Type:          api.LB_BACKEND_IP,
			BackendIP:     ips2[i].IPAddress,
			BackendPort:   self.DefaultPort,
		}

		ret = append(ret, &bg)
	}

	self.lbbs = ret
	return ret, nil
}

func (self *SLoadbalancerBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
	lbbs, err := self.GetILoadbalancerBackends()
	if err != nil {
		return nil, errors.Wrap(err, "GetILoadbalancerBackends")
	}

	for i := range lbbs {
		if lbbs[i].GetId() == backendId {
			return lbbs[i], nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetILoadbalancerBackendById")
}

func (self *SLoadbalancerBackendGroup) GetProtocolType() string {
	return ""
}

func (self *SLoadbalancerBackendGroup) GetScheduler() string {
	return ""
}

func (self *SLoadbalancerBackendGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	return nil, nil
}

func (self *SLoadbalancerBackendGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	return nil, nil
}

func (self *SLoadbalancerBackendGroup) AddBackendServer(serverId string, weight int, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "AddBackendServer")
}

func (self *SLoadbalancerBackendGroup) RemoveBackendServer(serverId string, weight int, port int) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "RemoveBackendServer")
}

func (self *SLoadbalancerBackendGroup) Delete(ctx context.Context) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadbalancerBackendGroup) Sync(ctx context.Context, group *cloudprovider.SLoadbalancerBackendGroup) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Sync")
}
