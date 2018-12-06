package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SSecurityGroupManager struct {
	ResourceManager
}

func NewSecurityGroupManager(regionId string, projectId string, signer auth.Signer) *SSecurityGroupManager {
	return &SSecurityGroupManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "security_group",
		KeywordPlural: "security_groups",

		ResourceKeyword: "security-groups",
	}}
}
