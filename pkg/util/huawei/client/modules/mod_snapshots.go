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
	return self.ListInContext(nil, "detail", querys)
}
