package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SElbWhitelistManager struct {
	SResourceManager
}

func NewElbWhitelistManager(regionId string, projectId string, signer auth.Signer, debug bool) *SElbWhitelistManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SElbWhitelistManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(signer, debug, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2.0",
		Keyword:       "whitelist",
		KeywordPlural: "whitelists",

		ResourceKeyword: "lbaas/whitelists",
	}}
}
