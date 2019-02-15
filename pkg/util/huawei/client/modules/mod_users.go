package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SUserManager struct {
	SResourceManager
}

func NewUserManager(signer auth.Signer, debug bool) *SUserManager {
	return &SUserManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3",
		Keyword:       "user",
		KeywordPlural: "users",

		ResourceKeyword: "users",
	}}
}
