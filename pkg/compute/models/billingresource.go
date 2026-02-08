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
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/billing"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SBillingTypeBase struct {
	// 计费类型, 按量、包年包月
	// example: prepaid, postpaid
	BillingType api.TBillingType `width:"36" charset:"ascii" nullable:"true" default:"postpaid" list:"user" create:"optional" json:"billing_type"`
}

type SBillingChargeTypeBase struct {
	// 计费类型: 流量、带宽
	// example: bandwidth
	ChargeType billing_api.TNetChargeType `width:"64" name:"charge_type" list:"user" create:"optional"`
}

type SBillingResourceBase struct {
	SBillingTypeBase

	// 包年包月到期时间
	ExpiredAt time.Time `nullable:"true" list:"user" json:"expired_at"`
	// 到期释放时间
	ReleaseAt time.Time `nullable:"true" list:"user" create:"optional" json:"release_at"`
	// 计费周期
	BillingCycle string `width:"10" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"billing_cycle"`
	// 是否自动续费
	AutoRenew bool `default:"false" list:"user" create:"optional" json:"auto_renew"`
}

type SBillingResourceBaseManager struct{}

func (self *SBillingResourceBase) GetChargeType() api.TBillingType {
	if len(self.BillingType) > 0 {
		return self.BillingType
	} else {
		return api.BILLING_TYPE_POSTPAID
	}
}

func (self *SBillingResourceBase) SetReleaseAt(releaseAt time.Time) {
	self.ReleaseAt = releaseAt
}

func (self *SBillingResourceBase) GetExpiredAt() time.Time {
	return self.ExpiredAt
}

func (self *SBillingResourceBase) GetReleaseAt() time.Time {
	return self.ReleaseAt
}

func (self *SBillingResourceBase) GetAutoRenew() bool {
	return self.AutoRenew
}

func (self *SBillingResourceBase) SetExpiredAt(expireAt time.Time) {
	self.ExpiredAt = expireAt
}

func (self *SBillingResourceBase) SetBillingCycle(billingCycle string) {
	self.BillingCycle = billingCycle
}

func (self *SBillingResourceBase) SetBillingType(billingType api.TBillingType) {
	self.BillingType = billingType
}

func (self *SBillingResourceBase) GetBillingType() api.TBillingType {
	return self.BillingType
}

func (self *SBillingResourceBase) getBillingBaseInfo() SBillingBaseInfo {
	info := SBillingBaseInfo{}
	info.ChargeType = self.GetChargeType()
	info.ExpiredAt = self.ExpiredAt
	info.ReleaseAt = self.ReleaseAt
	if self.GetChargeType() == api.BILLING_TYPE_PREPAID {
		info.BillingCycle = self.BillingCycle
	}
	return info
}

func (self *SBillingResourceBase) IsNotDeletablePrePaid() bool {
	if options.Options.PrepaidDeleteExpireCheck {
		return self.IsValidPrePaid()
	}

	return false
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
		if self.ReleaseAt.After(now) {
			return true
		}
	}
	return false
}

type SBillingBaseInfo struct {
	ChargeType api.TBillingType `json:",omitempty"`

	ExpiredAt    time.Time `json:",omitempty"`
	ReleaseAt    time.Time `json:",omitempty"`
	BillingCycle string    `json:",omitempty"`
}

type SCloudBillingInfo struct {
	SCloudProviderInfo

	SBillingBaseInfo

	PriceKey           string             `json:",omitempty"`
	InternetChargeType api.TNetChargeType `json:",omitempty"`
}

func (manager *SBillingResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.BillingDetailsInfo {
	rows := make([]api.BillingDetailsInfo, len(objs))
	for i := range rows {
		rows[i] = api.BillingDetailsInfo{}
	}
	return rows
}

func (manager *SBillingResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BillingResourceListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.BillingType) > 0 {
		if query.BillingType == api.BILLING_TYPE_POSTPAID {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(q.Field("billing_type")),
				sqlchemy.Equals(q.Field("billing_type"), api.BILLING_TYPE_POSTPAID),
			))
		} else {
			q = q.Equals("billing_type", api.BILLING_TYPE_PREPAID)
		}
	}
	if !query.BillingExpireBefore.IsZero() {
		q = q.LT("expired_at", query.BillingExpireBefore)
	}
	if !query.BillingExpireSince.IsZero() {
		q = q.GE("expired_at", query.BillingExpireBefore)
	}
	if len(query.BillingCycle) > 0 {
		q = q.Equals("billing_cycle", query.BillingCycle)
	}
	return q, nil
}

func (manager *SBillingResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BillingResourceListInput,
) (*sqlchemy.SQuery, error) {
	return q, nil
}

func ListExpiredPostpaidResources(
	q *sqlchemy.SQuery, limit int) *sqlchemy.SQuery {
	q = q.Equals("billing_type", api.BILLING_TYPE_POSTPAID)
	q = q.IsNotNull("release_at")
	q = q.LT("release_at", time.Now())
	if limit > 0 {
		q = q.Limit(limit)
	}
	return q
}
