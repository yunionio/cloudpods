package modules

import (
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/auth"
)

type SCredentialManager struct {
	SResourceManager
}

func NewCredentialManager(signer auth.Signer, debug bool) *SCredentialManager {
	return &SCredentialManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3.0",
		Keyword:       "credential",
		KeywordPlural: "credentials",

		ResourceKeyword: "OS-CREDENTIAL/credentials",
	}}
}
