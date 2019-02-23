package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/manager"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

// domian 客户账号ID https://support.huaweicloud.com/oce_faq/zh-cn_topic_0113714840.html
type SOrderManager struct {
	orderCtx manager.IManagerContext
	SResourceManager
}

type orderCtx struct {
	domainId string
}

// {domain_id}/common/
// 这个manager非常特殊。url hardcode
func (self *orderCtx) GetPath() string {
	return fmt.Sprintf("%s/common", self.domainId)
}

// 客户运营能力API的Endpoint为“bss.cn-north-1.myhuaweicloud.com”。该Endpoint为全局Endpoint，中国站所有区域均可使用。
// https://support.huaweicloud.com/api-oce/zh-cn_topic_0084961226.html
func NewOrderManager(signer auth.Signer, debug bool) *SOrderManager {
	return &SOrderManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameBSS,
		Region:        "cn-north-1",
		ProjectId:     "",
		version:       "v1.0",
		Keyword:       "",
		KeywordPlural: "",

		ResourceKeyword: "order-mgr",
	}}
}

func (self *SOrderManager) SetDomainId(domainId string) error {
	if len(domainId) == 0 {
		return fmt.Errorf("SetDomainId domain id should not be emtpy")
	}

	self.orderCtx = &orderCtx{domainId: domainId}
	return nil
}

// 查询客户包周期资源列表  https://support.huaweicloud.com/api-oce/zh-cn_topic_0084961226.html
func (self *SOrderManager) List(querys map[string]string) (*responses.ListResult, error) {
	return nil, fmt.Errorf("Not Suppport List Order")
}

// 查询订单的资源开通详情 https://support.huaweicloud.com/api-oce/api_order_00001.html
func (self *SOrderManager) Get(id string, querys map[string]string) (jsonutils.JSONObject, error) {
	if self.orderCtx == nil {
		return nil, fmt.Errorf("domainId is emtpy.Use SetDomainId method to set.")
	}

	// ！！！特殊调用
	return self.GetInContextWithSpec(self.orderCtx, "orders-resource", id, querys, "")
}

func (self *SOrderManager) PerformAction(action string, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	request := self.newRequest("POST", id, action, self.orderCtx)
	request.SetContent([]byte(getContent(params)))

	return self._do(request, "")
}

func (self *SOrderManager) GetPeriodResourceList(querys map[string]string) (*responses.ListResult, error) {
	if self.orderCtx == nil {
		return nil, fmt.Errorf("domainId is emtpy.Use SetDomainId method to set.")
	}

	return self.ListInContextWithSpec(self.orderCtx, "resources/detail", querys, "data")
}
