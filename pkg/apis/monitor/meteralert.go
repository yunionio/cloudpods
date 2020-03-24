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

package monitor

const (
	MeterAlertTypeBalance     = "balance"
	MeterAlertTypeDailyResFee = "resFee"
	MeterAlertTypeMonthResFee = "monthFee"
)

type MeterAlertCreateInput struct {
	ResourceAlertV1CreateInput

	// 监控资源类型, 比如: balance, resFree, monthFee
	Type string `json:"type"`
	// 云平台类型
	Provider string `json:"provider"`
	// 云账号 Id
	AccountId string `json:"account_id"`
	// 项目 Id string
	ProjectId string `json:"project_id"`
}

type MeterAlertListInput struct {
	V1AlertListInput

	// 监控资源类型, 比如: balance, resFree, monthFee
	Type string `json:"type"`
	// 云平台类型
	Provider string `json:"provider"`
	// 云账号 Id
	AccountId string `json:"account_id"`
	// 项目 Id
	ProjectId string `json:"project_id"`
}

type MeterAlertDetails struct {
	AlertV1Details

	Type      string `json:"type"`
	ProjectId string `json:"project_id"`
	AccountId string `json:"account_id"`
	Provider  string `json:"provider"`
}

type MeterAlertUpdateInput struct {
	V1AlertUpdateInput

	// 比较运算符, 比如: >, <, >=, <=
	Comparator *string `json:"comparator"`
	// 报警阀值
	Threshold *float64 `json:"threshold"`
	// 通知接受者
	Recipients *string `json:"recipients"`
	// 项目 Id
	ProjectId *string `json:"project_id"`
	// 通知方式, 比如: email, mobile
	Channel *string `json:"channel"`
	Status  *string `json:"status"`
}
