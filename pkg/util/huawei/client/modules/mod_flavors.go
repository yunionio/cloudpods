package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SFlavorManager struct {
	ResourceManager
}

func NewFlavorManager(regionId string, projectId string, signer auth.Signer) *SFlavorManager {
	return &SFlavorManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "flavor",
		KeywordPlural: "flavors",

		ResourceKeyword: "cloudservers/flavors", // 这个接口有点特殊，实际只用到了list一个方法。为了简便直接把cloudservers附上。
	}}
}
