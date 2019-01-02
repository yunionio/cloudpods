package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SBandwidthManager struct {
	ResourceManager
}

func NewBandwidthManager(regionId string, projectId string, signer auth.Signer) *SBandwidthManager {
	return &SBandwidthManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "bandwidth",
		KeywordPlural: "bandwidths",

		ResourceKeyword: "bandwidths",
	}}
}
