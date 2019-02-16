package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SSecurityGroupManager struct {
	SResourceManager
}

func NewSecurityGroupManager(regionId string, projectId string, signer auth.Signer, debug bool) *SSecurityGroupManager {
	return &SSecurityGroupManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "security_group",
		KeywordPlural: "security_groups",

		ResourceKeyword: "security-groups",
	}}
}

func NewNovaSecurityGroupManager(regionId string, projectId string, signer auth.Signer, debug bool) *SSecurityGroupManager {
	return &SSecurityGroupManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2.1",
		Keyword:       "security_group",
		KeywordPlural: "security_groups",

		ResourceKeyword: "os-security-groups",
	}}
}
