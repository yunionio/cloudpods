package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SDomainManager struct {
	SResourceManager
}

func NewDomainManager(signer auth.Signer, debug bool) *SDomainManager {
	return &SDomainManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3/auth",
		Keyword:       "domain",
		KeywordPlural: "domains",

		ResourceKeyword: "domains",
	}}
}
