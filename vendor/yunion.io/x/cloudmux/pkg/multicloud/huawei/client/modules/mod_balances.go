// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modules

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/manager"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/responses"
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
	SResourceManager
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
func NewBalanceManager(cfg manager.IManagerConfig) *SBalanceManager {
	return &SBalanceManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(cfg),
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
