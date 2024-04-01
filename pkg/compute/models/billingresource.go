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
	"fmt"
	"reflect"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/billing"
	notifyapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
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
	ResourceId   string `width:"128" charset:"ascii" primary:"true"`
	ResourceType string `width:"36" charset:"ascii" primary:"true"`
	AdvanceDays  int    `primary:"true"`
	LastCheck    time.Time
	NotifyNumber int
}

var BillingResourceCheckManager *SBillingResourceCheckManager

func init() {
	BillingResourceCheckManager = &SBillingResourceCheckManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SBillingResourceCheck{},
			"billingresourcecheck2_tbl",
			"billingresourcecheck",
			"billingresourcechecks",
		),
	}
	BillingResourceCheckManager.SetVirtualObject(BillingResourceCheckManager)
}

type IBillingModelManager interface {
	db.IModelManager
	GetExpiredModels(advanceDay int) ([]IBillingModel, error)
}

type IBillingModel interface {
	db.IModel
	GetExpiredAt() time.Time
}

func fetchExpiredModels(manager db.IModelManager, advanceDay int) ([]IBillingModel, error) {
	upLimit := time.Now().AddDate(0, 0, advanceDay+1)
	downLimit := time.Now().AddDate(0, 0, advanceDay)
	v := reflect.MakeSlice(reflect.SliceOf(manager.TableSpec().DataType()), 0, 0)
	q := manager.Query().LE("expired_at", upLimit).GE("expired_at", downLimit)

	vp := reflect.New(v.Type())
	vp.Elem().Set(v)
	err := db.FetchModelObjects(manager, q, vp.Interface())
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list %s", manager.KeywordPlural())
	}

	v = vp.Elem()
	log.Debugf("%s length of v: %d", manager.Alias(), v.Len())

	ms := make([]IBillingModel, v.Len())
	for i := range ms {
		ms[i] = v.Index(i).Addr().Interface().(IBillingModel)
	}
	return ms, nil
}

func (bm *SBillingResourceCheckManager) Create(ctx context.Context, resourceId, resourceType string, advanceDays int) error {
	bc := &SBillingResourceCheck{
		ResourceId:   resourceId,
		ResourceType: resourceType,
		AdvanceDays:  advanceDays,
		LastCheck:    time.Now(),
		NotifyNumber: 1,
	}
	bc.SetModelManager(bm, bc)
	return bm.TableSpec().InsertOrUpdate(ctx, bc)
}

func (bm *SBillingResourceCheckManager) Fetch(resourceIds []string, advanceDays int, length int) (map[string]*SBillingResourceCheck, error) {
	billingResourceChecks := make([]SBillingResourceCheck, 0, length)
	bq := bm.Query().Equals("advance_days", advanceDays).In("resource_id", resourceIds)
	err := db.FetchModelObjects(bm, bq, &billingResourceChecks)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]*SBillingResourceCheck, len(billingResourceChecks))
	for i := range billingResourceChecks {
		ret[billingResourceChecks[i].ResourceId] = &billingResourceChecks[i]
	}
	return ret, nil
}

func CheckBillingResourceExpireAt(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	billingResourceManagers := []IBillingModelManager{
		GuestManager,
		DBInstanceManager,
		ElasticcacheManager,
	}
	s := auth.GetAdminSession(ctx, options.Options.Region)
	resp, err := notify.NotifyTopic.List(s, jsonutils.Marshal(map[string]interface{}{
		"filter": fmt.Sprintf("name.equals('%s')", notifyapi.DefaultResourceRelease),
		"scope":  "system",
	}))
	if err != nil {
		log.Errorln(errors.Wrap(err, "list topics"))
		return
	}
	topics := []notifyapi.TopicDetails{}
	err = jsonutils.Update(&topics, resp.Data)
	if err != nil {
		log.Errorln(errors.Wrap(err, "update topic"))
		return
	}
	if len(topics) != 1 {
		log.Errorln(errors.Wrapf(errors.ErrNotSupported, "len topics :%d", len(topics)))
		return
	}

	for _, advanceDay := range topics[0].AdvanceDays {
		for _, manager := range billingResourceManagers {
			expiredModels, err := manager.GetExpiredModels(advanceDay)
			if err != nil {
				log.Errorf("unable to fetchExpiredModels: %s", err.Error())
				continue
			}
			mIds := make([]string, len(expiredModels))
			for i := range expiredModels {
				mIds[i] = expiredModels[i].GetId()
			}
			checks, err := BillingResourceCheckManager.Fetch(mIds, advanceDay, len(expiredModels))
			if err != nil {
				log.Errorf("unbale to fetch billingResourceChecks: %s", err.Error())
				continue
			}

			for i := range expiredModels {
				em := expiredModels[i]
				check, ok := checks[em.GetId()]
				if !ok {
					detailsDecro := func(ctx context.Context, details *jsonutils.JSONDict) {
						details.Set("advance_days", jsonutils.NewInt(int64(advanceDay)))
					}
					notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
						Obj:                 em,
						ObjDetailsDecorator: detailsDecro,
						Action:              notifyclient.ActionExpiredRelease,
						AdvanceDays:         advanceDay,
					})
					err := BillingResourceCheckManager.Create(ctx, em.GetId(), manager.Keyword(), advanceDay)
					if err != nil {
						log.Errorf("unable to create billingresourcecheck for resource %s %s", manager.Keyword(), em.GetId())
					}
					continue
				}
				if check.LastCheck.AddDate(0, 0, advanceDay).After(em.GetExpiredAt()) {
					continue
				}
				detailsDecro := func(ctx context.Context, details *jsonutils.JSONDict) {
					details.Set("advance_days", jsonutils.NewInt(int64(advanceDay)))
				}
				notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
					ObjDetailsDecorator: detailsDecro,
					Obj:                 em,
					Action:              notifyclient.ActionExpiredRelease,
					AdvanceDays:         advanceDay,
				})
				_, err := db.Update(check, func() error {
					check.LastCheck = time.Now()
					check.NotifyNumber += 1
					return nil
				})
				if err != nil {
					log.Errorf("unable to update billingresourcecheck for resource %s %s", manager.Keyword(), em.GetId())
				}
			}
		}
	}
}
