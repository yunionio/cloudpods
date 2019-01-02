package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SEipManager struct {
	ResourceManager
}

func NewEipManager(regionId string, projectId string, signer auth.Signer) *SEipManager {
	return &SEipManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "publicip",
		KeywordPlural: "publicips",

		ResourceKeyword: "publicips",
	}}
}
