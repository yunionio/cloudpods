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
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/requests"
)

type SPortManager struct {
	SResourceManager
}

type portProject struct {
	projectId string
}

// port接口查询时若非默认project，需要在header中指定X-Project-ID。url中未携带project信息(与其他接口相比有一点特殊)
// 绕过了ResourceManager中的projectid。直接在发送json请求前注入X-Project-ID
func (self *portProject) Process(request requests.IRequest) {
	request.AddHeaderParam("X-Project-Id", self.projectId)
}

func NewPortManager(regionId string, projectId string, signer auth.Signer, debug bool) *SPortManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SPortManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(signer, debug, &requestHook),
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "port",
		KeywordPlural: "ports",

		ResourceKeyword: "ports",
	}}
}
