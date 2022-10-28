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
	"yunion.io/x/log"

	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/manager"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/responses"
)

type SDiskManager struct {
	SResourceManager
}

func NewDiskManager(cfg manager.IManagerConfig) *SDiskManager {
	return &SDiskManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameEVS,
		Region:        cfg.GetRegionId(),
		ProjectId:     cfg.GetProjectId(),
		version:       "v2",
		Keyword:       "volume",
		KeywordPlural: "volumes",

		ResourceKeyword: "cloudvolumes",
	}}
}

func (self *SDiskManager) List(querys map[string]string) (*responses.ListResult, error) {
	return self.ListInContextWithSpec(nil, "detail", querys, self.KeywordPlural)
}

// https://support.huaweicloud.com/api-evs/evs_04_2003.html
func (self *SDiskManager) AsyncCreate(params jsonutils.JSONObject) (string, error) {
	origin_version := self.version
	self.version = "v2.1"
	defer func() { self.version = origin_version }()

	ret, err := self.CreateInContextWithSpec(nil, "", params, "")
	if err != nil {
		log.Debugf("AsyncCreate %s", err)
		return "", err
	}

	log.Debugf("AsyncCreate result %s", ret.String())
	// 按需机器
	jobId, err := ret.GetString("job_id")
	if err == nil {
		return jobId, nil
	}

	// 包年包月机器
	return ret.GetString("order_id")
}

// https://support.huaweicloud.com/api-evs/evs_04_2003.html
func (self *SDiskManager) GetDiskTypes() (*responses.ListResult, error) {
	originKeyword := self.ResourceKeyword
	self.ResourceKeyword = ""
	defer func() { self.ResourceKeyword = originKeyword }()
	return self.ListInContextWithSpec(self.ctx, "types", nil, "volume_types")
}
