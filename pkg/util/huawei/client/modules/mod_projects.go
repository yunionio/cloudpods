package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SProjectManager struct {
	ResourceManager
}

func NewProjectManager(signer auth.Signer) *SProjectManager {
	return &SProjectManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3",
		Keyword:       "project",
		KeywordPlural: "projects",

		ResourceKeyword: "projects",
	}}
}
