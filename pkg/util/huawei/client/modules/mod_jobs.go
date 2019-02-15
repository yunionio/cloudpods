package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

type SJobManager struct {
	SResourceManager
}

func NewJobManager(regionId string, projectId string, signer auth.Signer, debug bool) *SJobManager {
	return &SJobManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   "",
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "",
		KeywordPlural: "",

		ResourceKeyword: "jobs",
	}}
}

func (self *SJobManager) Get(id string, querys map[string]string) (jsonutils.JSONObject, error) {
	processedQuery, err := self.processQueryParam(querys)
	if err != nil {
		return nil, err
	}

	return self.GetInContext(nil, id, processedQuery)
}

func (self *SJobManager) List(querys map[string]string) (*responses.ListResult, error) {
	processedQuery, err := self.processQueryParam(querys)
	if err != nil {
		return nil, err
	}
	return self.ListInContext(nil, processedQuery)
}

// 兼容查询不同ServiceName服务的Job做的特殊处理。
func (self *SJobManager) processQueryParam(querys map[string]string) (map[string]string, error) {
	service_type, exists := querys["service_type"]
	if !exists {
		return querys, fmt.Errorf("must specific query parameter `service_type`. e.g. ecs|ims|iam")
	}

	self.ServiceName = ServiceNameType(service_type)
	delete(querys, "service_type")
	return querys, nil
}
