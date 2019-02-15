package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SBandwidthManager struct {
	SResourceManager
}

func NewBandwidthManager(regionId string, projectId string, signer auth.Signer, debug bool) *SBandwidthManager {
	return &SBandwidthManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "bandwidth",
		KeywordPlural: "bandwidths",

		ResourceKeyword: "bandwidths",
	}}
}
