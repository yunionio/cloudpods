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
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/manager"
)

type SFlavorManager struct {
	SResourceManager
}

func NewFlavorManager(cfg manager.IManagerConfig) *SFlavorManager {
	return &SFlavorManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameECS,
		Region:        cfg.GetRegionId(),
		ProjectId:     cfg.GetProjectId(),
		version:       "v1",
		Keyword:       "flavor",
		KeywordPlural: "flavors",

		ResourceKeyword: "cloudservers/flavors", // 这个接口有点特殊，实际只用到了list一个方法。为了简便直接把cloudservers附上。
	}}
}
