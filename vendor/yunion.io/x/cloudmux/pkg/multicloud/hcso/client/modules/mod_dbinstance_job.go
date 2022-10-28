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
	"yunion.io/x/jsonutils"

	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/manager"
)

type SDBInstanceJobManager struct {
	SResourceManager
}

func NewDBInstanceJobManager(cfg manager.IManagerConfig) *SDBInstanceJobManager {
	return &SDBInstanceJobManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameRDS,
		Region:        cfg.GetRegionId(),
		ProjectId:     cfg.GetProjectId(),
		version:       "v3",
		Keyword:       "",
		KeywordPlural: "",

		ResourceKeyword: "jobs",
	}}
}

func (self *SDBInstanceJobManager) Get(id string, querys map[string]string) (jsonutils.JSONObject, error) {
	return self.GetInContextWithSpec(nil, "", "", map[string]string{"id": id}, "job")
}
