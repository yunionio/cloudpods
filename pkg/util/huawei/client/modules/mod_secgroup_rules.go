package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SSecgroupRuleManager struct {
	ResourceManager
}

func NewSecgroupRuleManager(regionId string, projectId string, signer auth.Signer) *SSecgroupRuleManager {
	return &SSecgroupRuleManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "security_group_rule",
		KeywordPlural: "security_group_rules",

		ResourceKeyword: "security-group-rules",
	}}
}
