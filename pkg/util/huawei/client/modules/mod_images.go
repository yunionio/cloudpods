package modules

import (
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/requests"
)

type SImageManager struct {
	ResourceManager
}

type imageProject struct {
	projectId string
}

// image创建接口若非默认project，需要在header中指定X-Project-ID。url中未携带project信息(与其他接口相比有一点特殊)
// 绕过了ResourceManager中的projectid。直接在发送json请求前注入X-Project-ID
func (self *imageProject) Process(request requests.IRequest) {
	request.AddHeaderParam("X-Project-Id", self.projectId)
}

func NewImageManager(regionId string, projectId string, signer auth.Signer) *SImageManager {
	var requestHook imageProject
	if len(projectId) > 0 {
		requestHook = imageProject{projectId: projectId}
	}

	return &SImageManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}, requestHook: &requestHook},
		ServiceName:   ServiceNameIMS,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2",
		Keyword:       "image",
		KeywordPlural: "images",

		ResourceKeyword: "cloudimages",
	}}
}

func (self *SImageManager) Get(id string, querys map[string]string) (jsonutils.JSONObject, error) {
	if querys == nil {
		querys = make(map[string]string, 0)
	}

	querys["id"] = id
	// 这里默认使用private
	if t, exists := querys["__imagetype"]; !exists || len(t) == 0 {
		querys["__imagetype"] = "private"
	}

	ret, err := self.ListInContext(nil, querys)
	if err != nil {
		return nil, err
	}

	if ret.Data == nil || len(ret.Data) == 0 {
		return nil, httperrors.NewNotFoundError("image %s not found", id)
	}

	return ret.Data[0], nil
}

// https://support.huaweicloud.com/api-ims/zh-cn_topic_0020092108.html
// 删除image只能用这个manager
func NewOpenstackImageManager(regionId string, signer auth.Signer) *SImageManager {
	return &SImageManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameIMS,
		Region:        regionId,
		ProjectId:     "",
		version:       "v2",
		Keyword:       "image",
		KeywordPlural: "images",

		ResourceKeyword: "images",
	}}
}
