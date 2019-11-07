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
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLBListenerRule struct {
	listener *SLBListener

	Domain            string      `json:"Domain"`
	Certificate       certificate `json:"Certificate"`
	URL               string      `json:"Url"`
	HealthCheck       healthCheck `json:"HealthCheck"`
	LocationID        string      `json:"LocationId"`
	Scheduler         string      `json:"Scheduler"`
	SessionExpireTime int64       `json:"SessionExpireTime"`
}

// https://cloud.tencent.com/document/api/214/30688
func (self *SLBListenerRule) Delete() error {
	_, err := self.listener.lb.region.DeleteLBListenerRule(self.listener.lb.GetId(), self.listener.GetId(), self.GetId())
	if err != nil {
		return err
	}

	return cloudprovider.WaitDeleted(self, 5*time.Second, 60*time.Second)
}

func (self *SLBListenerRule) GetId() string {
	return self.LocationID
}

func (self *SLBListenerRule) GetName() string {
	return self.LocationID
}

func (self *SLBListenerRule) GetGlobalId() string {
	return self.LocationID
}

func (self *SLBListenerRule) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLBListenerRule) Refresh() error {
	err := self.listener.Refresh()
	if err != nil {
		return err
	}

	for _, rule := range self.listener.Rules {
		if rule.GetId() == self.GetId() {
			rule.listener = self.listener
			return jsonutils.Update(self, rule)
		}
	}

	return cloudprovider.ErrNotFound
}

func (self *SLBListenerRule) IsDefault() bool {
	return false
}

func (self *SLBListenerRule) IsEmulated() bool {
	return false
}

func (self *SLBListenerRule) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLBListenerRule) GetDomain() string {
	return self.Domain
}

func (self *SLBListenerRule) GetCondition() string {
	return ""
}

func (self *SLBListenerRule) GetPath() string {
	return self.URL
}

func (self *SLBListenerRule) GetProjectId() string {
	return ""
}

func (self *SLBListenerRule) GetBackendGroup() *SLBBackendGroup {
	t := self.listener.GetListenerType()
	if t == api.LB_LISTENER_TYPE_HTTP || t == api.LB_LISTENER_TYPE_HTTPS {
		return &SLBBackendGroup{
			lb:       self.listener.lb,
			listener: self.listener,
			rule:     self,
		}
	}

	return nil
}

// 只有http、https协议监听规则有backendgroupid
func (self *SLBListenerRule) GetBackendGroupId() string {
	bg := self.GetBackendGroup()
	if bg == nil {
		return ""
	}

	return bg.GetId()
}

// https://cloud.tencent.com/document/api/214/30688
// 返回requestId及error
func (self *SRegion) DeleteLBListenerRule(lbid, listenerId, ruleId string) (string, error) {
	if len(ruleId) == 0 {
		return "", fmt.Errorf("DeleteLBListenerRule rule id should not be empty")
	}
	return self.DeleteLBListenerRules(lbid, listenerId, []string{ruleId})
}

func (self *SRegion) DeleteLBListenerRules(lbid, listenerId string, ruleIds []string) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("DeleteLBListenerRules loadbalancer id should not be empty")
	}

	if len(listenerId) == 0 {
		return "", fmt.Errorf("DeleteLBListenerRules listener id should not be empty")
	}

	if len(ruleIds) == 0 {
		return "", fmt.Errorf("DeleteLBListenerRules rule id should not be empty")
	}

	params := map[string]string{"LoadBalancerId": lbid, "ListenerId": listenerId}
	for i, ruleId := range ruleIds {
		params[fmt.Sprintf("LocationIds.%d", i)] = ruleId
	}

	resp, err := self.clbRequest("DeleteRule", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}
