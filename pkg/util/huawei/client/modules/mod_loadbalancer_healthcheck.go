package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SElbHealthCheckManager struct {
	SResourceManager
}

func NewElbHealthCheckManager(regionId string, projectId string, signer auth.Signer, debug bool) *SElbHealthCheckManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SElbHealthCheckManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(signer, debug, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2.0",
		Keyword:       "healthmonitor",
		KeywordPlural: "healthmonitors",

		ResourceKeyword: "lbaas/healthmonitors",
	}}
}
