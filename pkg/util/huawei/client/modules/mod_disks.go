package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

type SDiskManager struct {
	SResourceManager
}

func NewDiskManager(regionId string, projectId string, signer auth.Signer, debug bool) *SDiskManager {
	return &SDiskManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
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
