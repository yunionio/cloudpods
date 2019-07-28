package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SLoadbalancerManager struct {
	SResourceManager
}

func NewLoadbalancerManager(regionId string, projectId string, signer auth.Signer, debug bool) *SLoadbalancerManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SLoadbalancerManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(signer, debug, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2.0",
		Keyword:       "loadbalancer",
		KeywordPlural: "loadbalancers",

		ResourceKeyword: "lbaas/loadbalancers",
	}}
}
