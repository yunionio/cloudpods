package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

type SDiskManager struct {
	ResourceManager
}

func NewDiskManager(regionId string, projectId string, signer auth.Signer) *SDiskManager {
	return &SDiskManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameEVS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "volume",
		KeywordPlural: "volumes",

		ResourceKeyword: "volumes",
	}}
}

func (self *SDiskManager) List(querys map[string]string) (*responses.ListResult, error) {
	return self.ListInContextWithSpec(nil, "detail", querys, self.KeywordPlural)
}
