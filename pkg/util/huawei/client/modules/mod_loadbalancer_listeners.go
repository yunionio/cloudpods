package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SElbListenersManager struct {
	SResourceManager
}

func NewElbListenersManager(regionId string, projectId string, signer auth.Signer, debug bool) *SElbListenersManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SElbListenersManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(signer, debug, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2.0",
		Keyword:       "listener",
		KeywordPlural: "listeners",

		ResourceKeyword: "lbaas/listeners",
	}}
}
