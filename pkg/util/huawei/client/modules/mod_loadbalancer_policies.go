package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SElbL7policiesManager struct {
	SResourceManager
}

func NewElbL7policiesManager(regionId string, projectId string, signer auth.Signer, debug bool) *SElbL7policiesManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SElbL7policiesManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(signer, debug, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2.0",
		Keyword:       "l7policy",
		KeywordPlural: "l7policies",

		ResourceKeyword: "lbaas/l7policies",
	}}
}
