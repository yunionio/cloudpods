package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SPortManager struct {
	ResourceManager
}

func NewPortManager(regionId string, signer auth.Signer) *SPortManager {
	return &SPortManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     "",
		version:       "v1",
		Keyword:       "port",
		KeywordPlural: "ports",

		ResourceKeyword: "ports",
	}}
}
