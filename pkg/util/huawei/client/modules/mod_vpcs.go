package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SVpcManager struct {
	SResourceManager
}

func NewVpcManager(regionId string, projectId string, signer auth.Signer, debug bool) *SVpcManager {
	return &SVpcManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "vpc",
		KeywordPlural: "vpcs",

		ResourceKeyword: "vpcs",
	}}
}
