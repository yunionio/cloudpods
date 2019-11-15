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

	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/responses"
)

type SElasticcacheManager struct {
	SResourceManager
}

type SDcsAvailableZoneManager struct {
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

func (self *SElasticcacheManager) Restart(instanceId string) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewArray(jsonutils.NewString(instanceId)), "instances")
	params.Add(jsonutils.NewString("restart"), "action")
	return self.UpdateInContextWithSpec(nil, "", "status", params, "")
}

// 当前版本，只有DCS2.0实例支持清空数据功能，即flush操作。
func (self *SElasticcacheManager) Flush(instanceId string) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewArray(jsonutils.NewString(instanceId)), "instances")
	params.Add(jsonutils.NewString("flush"), "action")
	return self.UpdateInContextWithSpec(nil, "", "status", params, "")
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423034.html
func (self *SElasticcacheManager) RestoreInstance(instanceId string, backupId string) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewArray(jsonutils.NewString(backupId)), "backup_id")

	return self.CreateInContextWithSpec(nil, fmt.Sprintf("%s/restores", instanceId), params, "")
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423024.html
func (self *SElasticcacheManager) ChangeInstanceSpec(instanceId string, specCode string, newCapacity int64) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("new_capacity", jsonutils.NewInt(newCapacity))
	params.Set("spec_code", jsonutils.NewString(specCode))

	return self.CreateInContextWithSpec(nil, fmt.Sprintf("%s/extend", instanceId), params, "")
}

func NewDcsAvailableZoneManager(regionId string, signer auth.Signer, debug bool) *SDcsAvailableZoneManager {
	return &SDcsAvailableZoneManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameDCS,
		Region:        regionId,
		ProjectId:     "",
		version:       "v1.0",
		Keyword:       "available_zone",
		KeywordPlural: "available_zones",

		ResourceKeyword: "availableZones",
	}}
}
