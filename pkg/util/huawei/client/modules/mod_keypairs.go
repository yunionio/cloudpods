package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SKeypairManager struct {
	ResourceManager
}

func NewKeypairManager(regionId string, projectId string, signer auth.Signer) *SKeypairManager {
	return &SKeypairManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "keypair",
		KeywordPlural: "keypairs",

		ResourceKeyword: "os-keypairs",
	}}
}
