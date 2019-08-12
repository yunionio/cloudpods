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

	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/responses"
)

type SElasticcacheManager struct {
	SResourceManager
}

func NewElasticcacheManager(regionId string, projectId string, signer auth.Signer, debug bool) *SElasticcacheManager {
	return &SElasticcacheManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameDCS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1.0",
		Keyword:       "",
		KeywordPlural: "instances",

		ResourceKeyword: "instances",
	}}
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423035.html
func (self *SElasticcacheManager) ListBackups(queries map[string]string) (*responses.ListResult, error) {
	var spec string
	if id, _ := queries["instance_id"]; len(id) == 0 {
		return nil, fmt.Errorf("SElasticcacheManager.ListBackups missing parameter instance_id")
	} else {
		spec = fmt.Sprintf("%s/backups", id)
	}

	delete(queries, "instance_id")
	return self.ListInContextWithSpec(nil, spec, queries, "backup_record_response")
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423027.html
func (self *SElasticcacheManager) ListParameters(queries map[string]string) (*responses.ListResult, error) {
	var spec string
	if id, _ := queries["instance_id"]; len(id) == 0 {
		return nil, fmt.Errorf("SElasticcacheManager.ListParameters missing parameter instance_id")
	} else {
		spec = fmt.Sprintf("%s/configs", id)
	}

	delete(queries, "instance_id")
	return self.ListInContextWithSpec(nil, spec, queries, "redis_config")
}
