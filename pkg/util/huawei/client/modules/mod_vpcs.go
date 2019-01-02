package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SVpcManager struct {
	ResourceManager
}

func NewVpcManager(regionId string, projectId string, signer auth.Signer) *SVpcManager {
	return &SVpcManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "vpc",
		KeywordPlural: "vpcs",

		ResourceKeyword: "vpcs",
	}}
}
