package modules

import (
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/requests"
)

type SPortManager struct {
	ResourceManager
}

type portProject struct {
	projectId string
}

// port接口查询时若非默认project，需要在header中指定X-Project-ID。url中未携带project信息(与其他接口相比有一点特殊)
// 绕过了ResourceManager中的projectid。直接在发送json请求前注入X-Project-ID
func (self *portProject) Process(request requests.IRequest) {
	request.AddHeaderParam("X-Project-Id", self.projectId)
}

func NewPortManager(regionId string, projectId string, signer auth.Signer) *SPortManager {
	var requestHook portProject
	if len(projectId) > 0 {
		requestHook = portProject{projectId: projectId}
	}

	return &SPortManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}, requestHook: &requestHook},
		ServiceName:   ServiceNameVPC,
		Region:        regionId,
		ProjectId:     "",
		version:       "v1",
		Keyword:       "port",
		KeywordPlural: "ports",

		ResourceKeyword: "ports",
	}}
}
