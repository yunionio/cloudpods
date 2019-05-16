package modules

import (
	"fmt"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
)

type SElbBackendManager struct {
	SResourceManager
}

type backendCtx struct {
	backendGroupId string
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561556.html
func (self *backendCtx) GetPath() string {
	return fmt.Sprintf("pools/%s", self.backendGroupId)
}

func NewElbBackendManager(regionId string, projectId string, signer auth.Signer, debug bool) *SElbBackendManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SElbBackendManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager2(signer, debug, &requestHook),
		ServiceName:   ServiceNameELB,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2.0/lbaas",
		Keyword:       "member",
		KeywordPlural: "members",

		ResourceKeyword: "members",
	}}
}

func (self *SElbBackendManager) SetBackendGroupId(lbgId string) error {
	if len(lbgId) == 0 {
		return fmt.Errorf("SetBackendGroupId id should not be emtpy")
	}

	self.ctx = &backendCtx{backendGroupId: lbgId}
	return nil
}
