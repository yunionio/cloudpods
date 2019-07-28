package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SElbCertificatesManager struct {
	SResourceManager
}

func NewElbCertificatesManager(regionId string, projectId string, signer auth.Signer, debug bool) *SElbCertificatesManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SElbCertificatesManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(signer, debug, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2.0",
		Keyword:       "",
		KeywordPlural: "certificates",

		ResourceKeyword: "lbaas/certificates",
	}}
}
