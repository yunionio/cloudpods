package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SSecgroupRuleManager struct {
	SResourceManager
}

func NewSecgroupRuleManager(regionId string, projectId string, signer auth.Signer, debug bool) *SSecgroupRuleManager {
	return &SSecgroupRuleManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "security_group_rule",
		KeywordPlural: "security_group_rules",

		ResourceKeyword: "security-group-rules",
	}}
}
