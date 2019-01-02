package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SInterfaceManager struct {
	ResourceManager
}

// 不建议使用
func NewInterfaceManager(regionId, projectId string, signer auth.Signer) *SInterfaceManager {
	return &SInterfaceManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "interfaceAttachment",
		KeywordPlural: "interfaceAttachments",

		ResourceKeyword: "os-interface",
	}}
}
