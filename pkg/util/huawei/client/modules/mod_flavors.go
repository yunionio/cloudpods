package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SFlavorManager struct {
	SResourceManager
}

func NewFlavorManager(regionId string, projectId string, signer auth.Signer, debug bool) *SFlavorManager {
	return &SFlavorManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "flavor",
		KeywordPlural: "flavors",

		ResourceKeyword: "cloudservers/flavors", // 这个接口有点特殊，实际只用到了list一个方法。为了简便直接把cloudservers附上。
	}}
}
