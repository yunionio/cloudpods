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

package qcloud

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLBBackend struct {
	group *SLBBackendGroup

	PublicIPAddresses  []string `json:"PublicIpAddresses"`
	Weight             int      `json:"Weight"`
	InstanceID         string   `json:"InstanceId"`
	InstanceName       string   `json:"InstanceName"`
	PrivateIPAddresses []string `json:"PrivateIpAddresses"`
	RegisteredTime     string   `json:"RegisteredTime"`
	Type               string   `json:"Type"`
	Port               int      `json:"Port"`
}

// ==========================================================
type SListenerBackend struct {
	Rules      []rule       `json:"Rules"`
	Targets    []SLBBackend `json:"Targets"`
	Protocol   string       `json:"Protocol"`
	ListenerID string       `json:"ListenerId"`
	Port       int64        `json:"Port"`
}

type rule struct {
	URL        string       `json:"Url"`
	Domain     string       `json:"Domain"`
	LocationID string       `json:"LocationId"`
	Targets    []SLBBackend `json:"Targets"`
}

// ==========================================================

// backend InstanceID + protocol  +Port + ip + rip全局唯一
func (self *SLBBackend) GetId() string {
	return fmt.Sprintf("%s/%s-%d", self.group.GetId(), self.InstanceID, self.Port)
}

func (self *SLBBackend) GetName() string {
	return self.GetId()
}

func (self *SLBBackend) GetGlobalId() string {
	return self.GetId()
}

func (self *SLBBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLBBackend) Refresh() error {
	backends, err := self.group.GetBackends()
	if err != nil {
		return err
	}

	for _, backend := range backends {
		if backend.GetId() == self.GetId() {
			return jsonutils.Update(self, backend)
		}
	}

	return cloudprovider.ErrNotFound
}

func (self *SLBBackend) IsEmulated() bool {
	return false
}

func (self *SLBBackend) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLBBackend) GetWeight() int {
	return self.Weight
}

func (self *SLBBackend) GetPort() int {
	return self.Port
}

func (self *SLBBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (self *SLBBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (self *SLBBackend) GetBackendId() string {
	return self.InstanceID
}

// 传统型： https://cloud.tencent.com/document/product/214/31790
func (self *SRegion) getClassicBackends(lbId, listenerId string) ([]SLBBackend, error) {
	params := map[string]string{"LoadBalancerId": lbId}

	resp, err := self.clbRequest("DescribeClassicalLBTargets", params)
	if err != nil {
		return nil, err
	}

	backends := []SLBBackend{}
	err = resp.Unmarshal(&backends, "Targets")
	if err != nil {
		return nil, err
	}
	return backends, nil
}

// 应用型： https://cloud.tencent.com/document/product/214/30684
func (self *SRegion) getBackends(lbId, listenerId, ruleId string) ([]SLBBackend, error) {
	params := map[string]string{"LoadBalancerId": lbId}

	if len(listenerId) > 0 {
		params["ListenerIds.0"] = listenerId
	}

	resp, err := self.clbRequest("DescribeTargets", params)
	if err != nil {
		return nil, err
	}

	lbackends := []SListenerBackend{}
	err = resp.Unmarshal(&lbackends, "Listeners")
	if err != nil {
		return nil, err
	}

	for _, entry := range lbackends {
		if (entry.Protocol == "HTTP" || entry.Protocol == "HTTPS") && len(ruleId) == 0 {
			return nil, fmt.Errorf("GetBackends for http/https listener %s must specific rule id", listenerId)
		}

		if len(ruleId) > 0 {
			for _, r := range entry.Rules {
				if r.LocationID == ruleId {
					return r.Targets, nil
				}
			}
		} else {
			return entry.Targets, nil
		}
	}

	// todo： 这里是返回空列表还是404？
	return []SLBBackend{}, nil
}

// 注意http、https监听器必须指定ruleId
func (self *SRegion) GetLBBackends(t LB_TYPE, lbId, listenerId, ruleId string) ([]SLBBackend, error) {
	if len(lbId) == 0 {
		return nil, fmt.Errorf("GetLBBackends loadbalancer id should not be empty")
	}

	if t == LB_TYPE_APPLICATION {
		return self.getBackends(lbId, listenerId, ruleId)
	} else if t == LB_TYPE_CLASSIC {
		return self.getClassicBackends(lbId, listenerId)
	} else {
		return nil, fmt.Errorf("GetLBBackends unsupported loadbalancer type %d", t)
	}
}

func (self *SLBBackend) GetProjectId() string {
	return ""
}

func (self *SLBBackend) SyncConf(port, weight int) error {
	err := self.group.UpdateBackendServer(self.InstanceID, self.Weight, self.Port, weight, port)
	if err != nil {
		return err
	}

	self.Port = port
	self.Weight = weight
	return nil
}
