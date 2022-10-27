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

	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/manager"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/responses"
)

type SJobManager struct {
	SResourceManager
}

func NewJobManager(cfg manager.IManagerConfig) *SJobManager {
	return &SJobManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   "",
		Region:        cfg.GetRegionId(),
		ProjectId:     cfg.GetProjectId(),
		version:       "v1",
		Keyword:       "",
		KeywordPlural: "",

		ResourceKeyword: "jobs",
	}}
}

func (self *SJobManager) Get(id string, querys map[string]string) (jsonutils.JSONObject, error) {
	processedQuery, err := self.processQueryParam(querys)
	if err != nil {
		return nil, err
	}

	return self.GetInContext(nil, id, processedQuery)
}

func (self *SJobManager) List(querys map[string]string) (*responses.ListResult, error) {
	processedQuery, err := self.processQueryParam(querys)
	if err != nil {
		return nil, err
	}
	return self.ListInContext(nil, processedQuery)
}

// 兼容查询不同ServiceName服务的Job做的特殊处理。
func (self *SJobManager) processQueryParam(querys map[string]string) (map[string]string, error) {
	service_type, exists := querys["service_type"]
	if !exists {
		return querys, fmt.Errorf("must specific query parameter `service_type`. e.g. ecs|ims|iam")
	}

	self.ServiceName = ServiceNameType(service_type)
	delete(querys, "service_type")
	return querys, nil
}
