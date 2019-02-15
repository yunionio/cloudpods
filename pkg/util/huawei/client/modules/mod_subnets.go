package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SSubnetManager struct {
	SResourceManager
}

func NewSubnetManager(regionId string, projectId string, signer auth.Signer, debug bool) *SSubnetManager {
	return &SSubnetManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "subnet",
		KeywordPlural: "subnets",

		ResourceKeyword: "subnets",
	}}
}
