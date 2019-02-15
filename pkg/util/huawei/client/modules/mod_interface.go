package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SInterfaceManager struct {
	SResourceManager
}

// 不建议使用
func NewInterfaceManager(regionId, projectId string, signer auth.Signer, debug bool) *SInterfaceManager {
	return &SInterfaceManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "interfaceAttachment",
		KeywordPlural: "interfaceAttachments",

		ResourceKeyword: "os-interface",
	}}
}
