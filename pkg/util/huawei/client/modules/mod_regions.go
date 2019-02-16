package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SRegionManager struct {
	SResourceManager
}

func NewRegionManager(signer auth.Signer, debug bool) *SRegionManager {
	return &SRegionManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3",
		Keyword:       "region",
		KeywordPlural: "regions",

		ResourceKeyword: "regions",
	}}
}
