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
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

type SSnapshotManager struct {
	SResourceManager
}

func NewSnapshotManager(regionId, projectId string, signer auth.Signer, debug bool) *SSnapshotManager {
	return &SSnapshotManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameEVS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "snapshot",
		KeywordPlural: "snapshots",

		ResourceKeyword: "snapshots",
	}}
}

func (self *SSnapshotManager) List(querys map[string]string) (*responses.ListResult, error) {
	return self.ListInContextWithSpec(nil, "detail", querys, self.KeywordPlural)
}

// https://support.huaweicloud.com/api-evs/zh-cn_topic_0051408629.html
// 回滚快照只能用这个manger。其他情况请不要使用
// 另外，香港-亚太还支持另外一个接口。https://support.huaweicloud.com/api-evs/zh-cn_topic_0142374138.html
func NewOsSnapshotManager(regionId string, projectId string, signer auth.Signer, debug bool) *SSnapshotManager {
	return &SSnapshotManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameEVS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "snapshot",
		KeywordPlural: "snapshots",

		ResourceKeyword: "os-vendor-snapshots",
	}}
}
