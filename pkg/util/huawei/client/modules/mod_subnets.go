package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SSubnetManager struct {
	ResourceManager
}

func NewSubnetManager(regionId string, projectId string, signer auth.Signer) *SSubnetManager {
	return &SSubnetManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "subnet",
		KeywordPlural: "subnets",

		ResourceKeyword: "subnets",
	}}
}
