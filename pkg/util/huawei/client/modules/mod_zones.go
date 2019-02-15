package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SZoneManager struct {
	SResourceManager
}

func NewZoneManager(regionId string, projectId string, signer auth.Signer, debug bool) *SZoneManager {
	return &SZoneManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "availabilityZoneInfo",
		KeywordPlural: "availabilityZoneInfo",

		ResourceKeyword: "os-availability-zone",
	}}
}
