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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/manager"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/requests"
)

type SImageManager struct {
	SResourceManager
}

type imageProject struct {
	projectId string
}

// image创建接口若非默认project，需要在header中指定X-Project-ID。url中未携带project信息(与其他接口相比有一点特殊)
// 绕过了ResourceManager中的projectid。直接在发送json请求前注入X-Project-ID
func (self *imageProject) Process(request requests.IRequest) {
	request.AddHeaderParam("X-Project-Id", self.projectId)
}

func NewImageManager(cfg manager.IManagerConfig) *SImageManager {
	var requestHook imageProject
	if len(cfg.GetProjectId()) > 0 {
		requestHook = imageProject{projectId: cfg.GetProjectId()}
	}

	return &SImageManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(cfg, &requestHook),
		ServiceName:   ServiceNameIMS,
		Region:        cfg.GetRegionId(),
		ProjectId:     "",
		version:       "v2",
		Keyword:       "image",
		KeywordPlural: "images",

		ResourceKeyword: "cloudimages",
	}}
}

//https://support.huaweicloud.com/api-ims/zh-cn_topic_0020091566.html
func (self *SImageManager) Get(id string, querys map[string]string) (jsonutils.JSONObject, error) {
	if querys == nil {
		querys = make(map[string]string, 0)
	}

	querys["id"] = id
	// 这里默认使用private
	// if t, exists := querys["__imagetype"]; !exists || len(t) == 0 {
	// 	querys["__imagetype"] = "private"
	// }

	ret, err := self.ListInContext(nil, querys)
	if err != nil {
		return nil, err
	}

	if ret.Data == nil || len(ret.Data) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "image %s not found", id)
	}

	return ret.Data[0], nil
}

// https://support.huaweicloud.com/api-ims/zh-cn_topic_0020092108.html
// 删除image只能用这个manager
func NewOpenstackImageManager(cfg manager.IManagerConfig) *SImageManager {
	var requestHook imageProject
	if len(cfg.GetProjectId()) > 0 {
		requestHook = imageProject{projectId: cfg.GetProjectId()}
	}

	return &SImageManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(cfg, &requestHook),
		ServiceName:   ServiceNameIMS,
		Region:        cfg.GetRegionId(),
		ProjectId:     "",
		version:       "v2",
		Keyword:       "image",
		KeywordPlural: "images",

		ResourceKeyword: "images",
	}}
}
