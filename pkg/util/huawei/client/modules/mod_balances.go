package modules

import (
	"fmt"
	"net/http"

	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

/*
https://support.huaweicloud.com/api-oce/zh-cn_topic_0075195195.html
客户运营能力API的Endpoint为“bss.cn-north-1.myhuaweicloud.com”。该Endpoint为全局Endpoint，中国站所有区域均可使用。
如何获取合作伙伴ID https://support.huaweicloud.com/bpconsole_faq/zh-cn_topic_0081005893.html
注意事项：
客户查询自身的账户余额的时候，只允许使用客户自身的AK/SK或者Token调用。
*/
type SBalanceManager struct {
	domainId string // 租户ID
	ResourceManager
}

type balanceCtx struct {
	domainId string
}

// https://support.huaweicloud.com/api-bpconsole/zh-cn_topic_0075213309.html
// 这个manager非常特殊。url hardcode
func (self *balanceCtx) GetPath() string {
	return fmt.Sprintf("%s/customer/account-mgr", self.domainId)
}

// 这个manager非常特殊。只有List	和 SetDomainId方法可用。其他方法未验证
func NewBalanceManager(signer auth.Signer) *SBalanceManager {
	return &SBalanceManager{ResourceManager: ResourceManager{
		BaseManager:   BaseManager{signer: signer, httpClient: &http.Client{}},
		ServiceName:   ServiceNameBSS,
		Region:        "cn-north-1",
		ProjectId:     "",
		version:       "v1.0",
		Keyword:       "account_balance",
		KeywordPlural: "account_balances",

		ResourceKeyword: "balances",
	}}
}

func (self *SBalanceManager) List(querys map[string]string) (*responses.ListResult, error) {
	if len(self.domainId) == 0 {
		return nil, fmt.Errorf("domainId is emtpy.Use SetDomainId method to set.")
	}

	ctx := &balanceCtx{domainId: self.domainId}
	return self.ListInContext(ctx, querys)
}

func (self *SBalanceManager) SetDomainId(domainId string) {
	self.domainId = domainId
}
