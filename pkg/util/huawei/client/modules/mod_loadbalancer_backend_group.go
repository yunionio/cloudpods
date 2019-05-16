package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SElbBackendGroupManager struct {
	SResourceManager
}

func NewElbBackendGroupManager(regionId string, projectId string, signer auth.Signer, debug bool) *SElbBackendGroupManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SElbBackendGroupManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(signer, debug, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2.0",
		Keyword:       "pool",
		KeywordPlural: "pools",

		ResourceKeyword: "lbaas/pools",
	}}
}
