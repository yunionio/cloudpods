package modules

import (
	"net/http"
	"strconv"

	"fmt"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

type SServerManager struct {
	ResourceManager
}

func NewServerManager(regionId, projectId string, signer auth.Signer) *SServerManager {
	return &SServerManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameECS,
		Region:        regionId,
		ProjectId:     projectId,
		version:       "v1",
		Keyword:       "server",
		KeywordPlural: "servers",

		ResourceKeyword: "cloudservers",
	}}
}

func (self *SServerManager) List(querys map[string]string) (*responses.ListResult, error) {
	if offset, exists := querys["offset"]; !exists {
		// 华为云分页参数各式各样。cloudserver offset从1开始。部分其他接口从0开始。
		// 另外部分接口使用pager分页 或者 maker分页
		querys["offset"] = "1"
	} else {
		n, err := strconv.Atoi(offset)
		if err != nil {
			return nil, fmt.Errorf("offset is invalid: %s", offset)
		}
		querys["offset"] = strconv.Itoa(n + 1)
	}
	return self.ListInContext(nil, "detail", querys)
}
