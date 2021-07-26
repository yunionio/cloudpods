package modules

import (
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/manager"
)

type SCredentialManager struct {
	SResourceManager
}

func NewCredentialManager(cfg manager.IManagerConfig) *SCredentialManager {
	return &SCredentialManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3.0",
		Keyword:       "credential",
		KeywordPlural: "credentials",

		ResourceKeyword: "OS-CREDENTIAL/credentials",
	}}
}
