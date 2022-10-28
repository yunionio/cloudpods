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

type SElbBackendManager struct {
	SResourceManager
}

type backendCtx struct {
	backendGroupId string
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561556.html
func (self *backendCtx) GetPath() string {
	return fmt.Sprintf("pools/%s", self.backendGroupId)
}

func NewElbBackendManager(cfg manager.IManagerConfig) *SElbBackendManager {
	var requestHook portProject
	if len(cfg.GetProjectId()) > 0 {
		requestHook = portProject{projectId: cfg.GetProjectId()}
	}

	return &SElbBackendManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(cfg, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        cfg.GetRegionId(),
		ProjectId:     "",
		version:       "v2.0/lbaas",
		Keyword:       "member",
		KeywordPlural: "members",

		ResourceKeyword: "members",
	}}
}

func (self *SElbBackendManager) SetBackendGroupId(lbgId string) error {
	if len(lbgId) == 0 {
		return fmt.Errorf("SetBackendGroupId id should not be emtpy")
	}

	self.ctx = &backendCtx{backendGroupId: lbgId}
	return nil
}
