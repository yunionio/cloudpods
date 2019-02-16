package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SProjectManager struct {
	SResourceManager
}

func NewProjectManager(signer auth.Signer, debug bool) *SProjectManager {
	return &SProjectManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3",
		Keyword:       "project",
		KeywordPlural: "projects",

		ResourceKeyword: "projects",
	}}
}
