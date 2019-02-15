package modules

import (
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SKeypairManager struct {
	SResourceManager
}

func NewKeypairManager(regionId string, projectId string, signer auth.Signer, debug bool) *SKeypairManager {
	return &SKeypairManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v2",
		Keyword:       "keypair",
		KeywordPlural: "keypairs",

		ResourceKeyword: "os-keypairs",
	}}
}
