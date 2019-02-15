package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SEipManager struct {
	SResourceManager
}

func NewEipManager(regionId string, projectId string, signer auth.Signer, debug bool) *SEipManager {
	return &SEipManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "publicip",
		KeywordPlural: "publicips",

		ResourceKeyword: "publicips",
	}}
}
