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

package modules

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/manager"
)

type SElbPoliciesManager struct {
	SResourceManager
}

type policyCtx struct {
	l7policyId string
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561556.html
func (self *policyCtx) GetPath() string {
	return fmt.Sprintf("l7policies/%s", self.l7policyId)
}

func NewElbPoliciesManager(cfg manager.IManagerConfig) *SElbPoliciesManager {
	var requestHook portProject
	if len(cfg.GetProjectId()) > 0 {
		requestHook = portProject{projectId: cfg.GetProjectId()}
	}

	return &SElbPoliciesManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(cfg, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        cfg.GetRegionId(),
		ProjectId:     "",
		version:       "v2.0/lbaas",
		Keyword:       "rule",
		KeywordPlural: "rules",

		ResourceKeyword: "rules",
	}}
}

func (self *SElbPoliciesManager) SetL7policyId(lbpId string) error {
	if len(lbpId) == 0 {
		return fmt.Errorf("SetL7policyId id should not be emtpy")
	}

	self.ctx = &policyCtx{l7policyId: lbpId}
	return nil
}
