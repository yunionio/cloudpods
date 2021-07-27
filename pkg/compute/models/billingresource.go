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
	"reflect"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/billing"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SBillingResourceBase struct {
	// 计费类型, 按量、包年包月
	// example: postpaid
	BillingType string `width:"36" charset:"ascii" nullable:"true" default:"postpaid" list:"user" create:"optional" json:"billing_type"`
	// 过期时间
	ExpiredAt time.Time `nullable:"true" list:"user" create:"optional" json:"expired_at"`
	// 计费周期
	BillingCycle string `width:"10" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"billing_cycle"`
	// 是否自动续费
	AutoRenew bool `default:"false" list:"user" create:"optional" json:"auto_renew"`
}

type SBillingResourceBaseManager struct{}

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
	info.ExpiredAt = self.ExpiredAt
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
	SCloudProviderInfo

	SBillingBaseInfo

	PriceKey           string `json:",omitempty"`
	InternetChargeType string `json:",omitempty"`
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
	q = q.IsNotNull("expired_at")
	q = q.LT("expired_at", time.Now())
	if limit > 0 {
		q = q.Limit(limit)
	}
	return q
}

func ParseBillingCycleInput(billingBase *SBillingResourceBase, input apis.PostpaidExpireInput) (*billing.SBillingCycle, error) {
	var (
		bc          billing.SBillingCycle
		err         error
		durationStr string
	)
	if len(input.Duration) == 0 {
		if input.ExpireTime.IsZero() {
			return nil, httperrors.NewInputParameterError("missing duration/expire_time")
		}
		timeC := billingBase.ExpiredAt
		if timeC.IsZero() {
			timeC = time.Now()
		}
		dur := input.ExpireTime.Sub(timeC)
		if dur <= 0 {
			return nil, httperrors.NewInputParameterError("expire time is before current expire at")
		}
		bc = billing.DurationToBillingCycle(dur)
	} else {
		bc, err = billing.ParseBillingCycle(durationStr)
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid duration %s: %s", durationStr, err)
		}
	}

	return &bc, nil
}

type SBillingResourceCheckManager struct {
	db.SResourceBaseManager
}

type SBillingResourceCheck struct {
	db.SResourceBase
	ResourceId   string `width:"128" charset:"ascii" index:"true"`
	ResourceType string `width:"36" charset:"ascii" index:"true"`
	AdvanceDays  int
	LastCheck    time.Time
}

var BillingResourceCheckManager *SBillingResourceCheckManager

func init() {
	BillingResourceCheckManager = &SBillingResourceCheckManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SBillingResourceCheck{},
			"billingresourcecheck_tbl",
			"billingresourcecheck",
			"billingresourcechecks",
		),
	}
	BillingResourceCheckManager.SetVirtualObject(BillingResourceCheckManager)
}

func (bm *SBillingResourceCheckManager) Create(ctx context.Context, resourceId, resourceType string, advanceDays int) error {
	bc := SBillingResourceCheck{
		ResourceId:   resourceId,
		ResourceType: resourceType,
		AdvanceDays:  advanceDays,
		LastCheck:    time.Now(),
	}
	return bm.TableSpec().Insert(ctx, &bc)
}

var advanceDays []int = []int{1, 3}

func CheckBillingResourceExpireAt(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	billingResourceManagers := []db.IModelManager{
		GuestManager,
		DBInstanceManager,
		ElasticcacheManager,
	}
	for _, advanceDay := range advanceDays {
		for _, manager := range billingResourceManagers {
			upLimit := time.Now().AddDate(0, 0, advanceDay)
			downLimit := time.Now().AddDate(0, 0, advanceDay-1)
			v := reflect.MakeSlice(reflect.SliceOf(manager.TableSpec().DataType()), 0, 0)
			q := manager.Query().LE("expired_at", upLimit).GE("expired_at", downLimit)

			bq := BillingResourceCheckManager.Query("resource_id").Equals("resource_type", manager.Keyword()).Equals("advance_days", advanceDay).SubQuery()
			q = q.LeftJoin(bq, sqlchemy.Equals(q.Field("id"), bq.Field("resource_id")))
			q = q.IsNull("resource_id")

			vp := reflect.New(v.Type())
			vp.Elem().Set(v)
			err := db.FetchModelObjects(manager, q, vp.Interface())
			if err != nil {
				log.Errorf("unable to list %s: %v", manager.KeywordPlural(), err)
			}

			v = vp.Elem()
			log.Debugf("%s length of v: %d", manager.Alias(), v.Len())
			for i := 0; i < v.Len(); i++ {
				m := v.Index(i).Addr().Interface().(db.IModel)
				notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
					Obj:         m,
					Action:      notifyclient.ActionExpiredRelease,
					AdvanceDays: advanceDay,
				})
				err := BillingResourceCheckManager.Create(ctx, m.GetId(), manager.Keyword(), advanceDay)
				if err != nil {
					log.Errorf("unable to create billingresourcecheck for resource %s %s", manager.Keyword(), m.GetId())
				}
			}
		}
	}
}
