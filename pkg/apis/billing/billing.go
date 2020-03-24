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

package billing

import "time"

type BillingDetailsInfo struct {
}

type BillingResourceListInput struct {
	// 计费类型，按需计费和预付费
	// pattern:prepaid|postpaid
	BillingType string `json:"billing_type"`

	// 计费过期时间的查询起始时间
	BillingExpireSince time.Time `json:"billing_expire_since"`
	// 计费过期时间的查询终止时间
	BillingExpireBefore time.Time `json:"billing_expire_before"`

	// 计费周期
	// example:7d
	BillingCycle string `json:"billing_cycle"`
}
