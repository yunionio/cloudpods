package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SUserManager struct {
	ResourceManager
}

func NewUserManager(signer auth.Signer) *SUserManager {
	return &SUserManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3",
		Keyword:       "user",
		KeywordPlural: "users",

		ResourceKeyword: "users",
	}}
}
