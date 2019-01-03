package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

type SSnapshotManager struct {
	ResourceManager
}

func NewSnapshotManager(regionId, projectId string, signer auth.Signer) *SSnapshotManager {
	return &SSnapshotManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
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
func NewOsSnapshotManager(regionId string, projectId string, signer auth.Signer) *SSnapshotManager {
	return &SSnapshotManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameEVS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "snapshot",
		KeywordPlural: "snapshots",

		ResourceKeyword: "os-vendor-snapshots",
	}}
}
