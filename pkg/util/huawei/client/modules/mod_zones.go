package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SZoneManager struct {
	ResourceManager
}

func NewZoneManager(regionId string, projectId string, signer auth.Signer) *SZoneManager {
	return &SZoneManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "availabilityZoneInfo",
		KeywordPlural: "availabilityZoneInfo",

		ResourceKeyword: "os-availability-zone",
	}}
}
