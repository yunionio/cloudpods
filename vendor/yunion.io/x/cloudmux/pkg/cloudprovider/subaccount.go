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

package cloudprovider

import (
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

type SSubAccount struct {
	Id string // 账号ID
	// 若Account不为空，可不传
	Name string
	// 描述信息
	Desc string
	// 输入必填，若为空，需要指定子账号名称
	Account          string
	HealthStatus     string // 云端服务健康状态。例如欠费、项目冻结都属于不健康状态。
	DefaultProjectId string // 默认云订阅项目Id
}

// +onecloud:model-api-gen
type SubAccounts struct {
	// 若输出则是全量子账号列表，若输入，代表允许同步的子账号
	Accounts []SSubAccount
	// 若输出是云账号查询到的区域列表，若输入，代表允许同步的区域
	Cloudregions []struct {
		// 输入必填
		Id     string
		Name   string
		Status string
	}
}

func (self SubAccounts) IsZero() bool {
	return len(self.Accounts) == 0
}

func (self SubAccounts) String() string {
	return jsonutils.Marshal(self).String()
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SubAccounts{}), func() gotypes.ISerializable {
		return &SubAccounts{}
	})
}
