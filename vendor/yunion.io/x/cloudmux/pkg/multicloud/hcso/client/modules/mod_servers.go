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
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/manager"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/responses"
)

type SServerManager struct {
	SResourceManager
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212668.html
// v.1.1 新增支持创建包年/包月的弹性云服务器。！！但是不支持查询等调用 https://support.huaweicloud.com/api-ecs/zh-cn_topic_0093055772.html
func NewServerManager(cfg manager.IManagerConfig) *SServerManager {
	return &SServerManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameECS,
		Region:        cfg.GetRegionId(),
		ProjectId:     cfg.GetProjectId(),
		version:       "v1",
		Keyword:       "server",
		KeywordPlural: "servers",

		ResourceKeyword: "cloudservers",
	}}
}

func (self *SServerManager) List(querys map[string]string) (*responses.ListResult, error) {
	if offset, exists := querys["offset"]; !exists {
		// 华为云分页参数各式各样。cloudserver offset从1开始。部分其他接口从0开始。
		// 另外部分接口使用pager分页 或者 maker分页
		querys["offset"] = "1"
	} else {
		n, err := strconv.Atoi(offset)
		if err != nil {
			return nil, fmt.Errorf("offset is invalid: %s", offset)
		}
		querys["offset"] = strconv.Itoa(n + 1)
	}
	return self.ListInContextWithSpec(nil, "detail", querys, self.KeywordPlural)
}

/*
返回job id 或者 order id

https://support.huaweicloud.com/api-ecs/zh-cn_topic_0093055772.html
创建按需的弹性云服务 ——> job_id 任务ID （返回数据uuid举例："70a599e0-31e7-49b7-b260-868f441e862b"）
包年包月机器  --> order_id (返回数据举例： "CS1711152257C60TL")
*/
func (self *SServerManager) AsyncCreate(params jsonutils.JSONObject) (string, error) {
	origin_version := self.version
	self.version = "v1.1"
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

func (self *SServerManager) Create(params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("not supported.please use AsyncCreate")
}

// 不推荐使用这个manager
func NewNovaServerManager(cfg manager.IManagerConfig) *SServerManager {
	return &SServerManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameECS,
		Region:        cfg.GetRegionId(),
		ProjectId:     cfg.GetProjectId(),
		version:       "v2.1",
		Keyword:       "server",
		KeywordPlural: "servers",

		ResourceKeyword: "servers",
	}}
}

// 重装弹性云服务器操作系统（安装Cloud-init）,请用这个manager
func NewServerV2Manager(cfg manager.IManagerConfig) *SServerManager {
	return &SServerManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameECS,
		Region:        cfg.GetRegionId(),
		ProjectId:     cfg.GetProjectId(),
		version:       "v2",
		Keyword:       "server",
		KeywordPlural: "servers",

		ResourceKeyword: "cloudservers",
	}}
}
