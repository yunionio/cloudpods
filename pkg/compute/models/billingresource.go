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

package models

import (
	"time"

	api "yunion.io/x/onecloud/pkg/apis/billing"
	"yunion.io/x/onecloud/pkg/apis/compute"
)

type SBillingResourceBase struct {
	// 计费类型, 按量、包年包月
	// example: postpaid
	BillingType string `width:"36" charset:"ascii" nullable:"true" default:"postpaid" list:"user" create:"optional"`
	// 过期时间
	ExpiredAt time.Time `nullable:"true" list:"user" create:"optional"`
	// 计费周期
	BillingCycle string `width:"10" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

func (self *SBillingResourceBase) GetChargeType() string {
	if len(self.BillingType) > 0 {
		return self.BillingType
	} else {
		return api.BILLING_TYPE_POSTPAID
	}
}

func (self *SBillingResourceBase) getBillingBaseInfo() SBillingBaseInfo {
	info := SBillingBaseInfo{}
	info.ChargeType = self.GetChargeType()
	if self.GetChargeType() == api.BILLING_TYPE_PREPAID {
		info.ExpiredAt = self.ExpiredAt
		info.BillingCycle = self.BillingCycle
	}
	return info
}

func (self *SBillingResourceBase) IsValidPrePaid() bool {
	if self.BillingType == api.BILLING_TYPE_PREPAID {
		now := time.Now().UTC()
		if self.ExpiredAt.After(now) {
			return true
		}
	}
	return false
}

func (self *SBillingResourceBase) IsValidPostPaid() bool {
	if self.BillingType == api.BILLING_TYPE_POSTPAID {
		now := time.Now().UTC()
		if self.ExpiredAt.After(now) {
			return true
		}
	}
	return false
}

type SBillingBaseInfo struct {
	ChargeType   string    `json:",omitempty"`
	ExpiredAt    time.Time `json:",omitempty"`
	BillingCycle string    `json:",omitempty"`
}

type SCloudBillingInfo struct {
	compute.CloudproviderInfo

	SBillingBaseInfo

	PriceKey           string `json:",omitempty"`
	InternetChargeType string `json:",omitempty"`
}
