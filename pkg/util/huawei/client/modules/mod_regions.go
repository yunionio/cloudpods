package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SRegionManager struct {
	ResourceManager
}

func NewRegionManager(signer auth.Signer) *SRegionManager {
	return &SRegionManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3",
		Keyword:       "region",
		KeywordPlural: "regions",

		ResourceKeyword: "regions",
	}}
}
