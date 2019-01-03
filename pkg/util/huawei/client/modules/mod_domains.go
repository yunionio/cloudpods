package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SDomainManager struct {
	ResourceManager
}

func NewDomainManager(signer auth.Signer) *SDomainManager {
	return &SDomainManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3/auth",
		Keyword:       "domain",
		KeywordPlural: "domains",

		ResourceKeyword: "domains",
	}}
}
