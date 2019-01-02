package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SImageManager struct {
	ResourceManager
}

func NewImageManager(regionId string, signer auth.Signer) *SImageManager {
	return &SImageManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameIMS,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2",
		Keyword:       "image",
		KeywordPlural: "images",

		ResourceKeyword: "cloudimages",
	}}
}
