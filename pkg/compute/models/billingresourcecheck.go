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

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	notifyapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SBillingResourceCheckManager struct {
	db.SVirtualResourceBaseManager
}

var BillingResourceCheckManager *SBillingResourceCheckManager

func init() {
	BillingResourceCheckManager = &SBillingResourceCheckManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SBillingResourceCheck{},
			"billing_resource_checks_tbl",
			"billing_resource_check",
			"billing_resource_checks",
		),
	}
	BillingResourceCheckManager.SetVirtualObject(BillingResourceCheckManager)
}

type SBillingResourceCheck struct {
	db.SVirtualResourceBase
	SBillingResourceBase

	ResourceType string `width:"36" charset:"ascii" list:"user"`
}

type IBillingModelManager interface {
	db.IModelManager
	GetExpiredModels(advanceDay int) ([]IBillingModel, error)
}

type IBillingModel interface {
	db.IModel
	GetOwnerId() mcclient.IIdentityProvider
	GetStatus() string
	GetExpiredAt() time.Time
	GetReleaseAt() time.Time
	GetAutoRenew() bool
	SetReleaseAt(releaseAt time.Time)
	SetExpiredAt(expireAt time.Time)
	SetBillingCycle(billingCycle string)
	SetBillingType(billingType string)
	GetBillingType() string
}

// 即将到期释放资源列表
func (manager *SBillingResourceCheckManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.BillingResourceCheckListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	if len(query.ResourceType) > 0 {
		q = q.In("resource_type", query.ResourceType)
	}
	return q, nil
}

func SaveReleaseAt(ctx context.Context, model IBillingModel, userCred mcclient.TokenCredential, releaseAt time.Time) error {
	diff, err := db.Update(model, func() error {
		model.SetReleaseAt(releaseAt)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	if len(diff) > 0 {
		db.OpsLog.LogEvent(model, db.ACT_SET_RELEASE_TIME, fmt.Sprintf("release at: %s", releaseAt), userCred)
	}
	if len(diff) > 0 && userCred != nil {
		logclient.AddActionLogWithContext(ctx, model, logclient.ACT_SET_RELEASE_TIME, diff, userCred, true)
	}
	return nil
}

func SaveRenewInfo(
	ctx context.Context, userCred mcclient.TokenCredential,
	model IBillingModel, bc *billing.SBillingCycle, expireAt *time.Time, billingType string,
) error {
	_, err := db.Update(model, func() error {
		if billingType == "" {
			billingType = billing_api.BILLING_TYPE_PREPAID
		}
		if model.GetBillingType() == "" {
			model.SetBillingType(billingType)
		}
		if expireAt != nil && !expireAt.IsZero() {
			model.SetExpiredAt(*expireAt)
		} else if bc != nil {
			model.SetBillingCycle(bc.String())
			model.SetExpiredAt(bc.EndAt(model.GetExpiredAt()))
		}
		return nil
	})
	if err != nil {
		log.Errorf("UpdateItem error %s", err)
		return err
	}
	db.OpsLog.LogEvent(model, db.ACT_RENEW, model.GetShortDesc(ctx), userCred)
	return nil
}

func (manager *SBillingResourceCheckManager) GetExpiredModels(advanceDay int) ([]SBillingResourceCheck, error) {
	upLimit := time.Now().AddDate(0, 0, advanceDay+1)
	downLimit := time.Now().AddDate(0, 0, advanceDay)
	q := manager.Query()
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("billing_type"), billing_api.BILLING_TYPE_POSTPAID),
				sqlchemy.LE(q.Field("release_at"), upLimit),
				sqlchemy.GE(q.Field("release_at"), downLimit),
			),
			// 跳过自动续费实例
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("billing_type"), billing_api.BILLING_TYPE_PREPAID),
				sqlchemy.LE(q.Field("expired_at"), upLimit),
				sqlchemy.GE(q.Field("expired_at"), downLimit),
				sqlchemy.IsFalse(q.Field("auto_renew")),
			)))

	ret := []SBillingResourceCheck{}
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}
	return ret, nil
}

func fetchExpiredModels(manager db.IModelManager, advanceDay int) ([]IBillingModel, error) {
	upLimit := time.Now().AddDate(0, 0, advanceDay+1)
	downLimit := time.Now()
	v := reflect.MakeSlice(reflect.SliceOf(manager.TableSpec().DataType()), 0, 0)
	q := manager.Query()
	q = q.Filter(
		sqlchemy.OR(
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("billing_type"), billing_api.BILLING_TYPE_POSTPAID),
				sqlchemy.LE(q.Field("release_at"), upLimit),
				sqlchemy.GE(q.Field("release_at"), downLimit),
			),
			sqlchemy.AND(
				sqlchemy.Equals(q.Field("billing_type"), billing_api.BILLING_TYPE_PREPAID),
				sqlchemy.LE(q.Field("expired_at"), upLimit),
				sqlchemy.GE(q.Field("expired_at"), downLimit),
			)))

	vp := reflect.New(v.Type())
	vp.Elem().Set(v)
	err := db.FetchModelObjects(manager, q, vp.Interface())
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list %s", manager.KeywordPlural())
	}

	v = vp.Elem()
	if v.Len() > 0 {
		log.Debugf("%s length of v: %d will be notified", manager.Alias(), v.Len())
	}

	ms := make([]IBillingModel, v.Len())
	for i := range ms {
		ms[i] = v.Index(i).Addr().Interface().(IBillingModel)
	}
	return ms, nil
}

func (bm *SBillingResourceCheckManager) Create(ctx context.Context, res IBillingModel, resourceType string) error {
	bc := &SBillingResourceCheck{
		ResourceType: resourceType,
	}
	bc.Id = res.GetId()
	bc.Name = res.GetName()
	bc.ExpiredAt = res.GetExpiredAt()
	bc.ReleaseAt = res.GetReleaseAt()
	bc.AutoRenew = res.GetAutoRenew()
	bc.BillingType = res.GetBillingType()
	if owner := res.GetOwnerId(); owner != nil {
		bc.ProjectId = owner.GetProjectId()
		bc.DomainId = owner.GetDomainId()
	}
	bc.Status = res.GetStatus()
	bc.SetModelManager(bm, bc)
	return bm.TableSpec().InsertOrUpdate(ctx, bc)
}

func (man *SBillingResourceCheckManager) PerformCheck(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	scanExpiredBillingResources(ctx)
	return jsonutils.NewDict(), nil
}

func (man *SBillingResourceCheckManager) clean(resType string, resourceIds []string) error {
	ids, err := db.FetchField(man, "id", func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("resource_type", resType).NotIn("id", resourceIds)
	})
	if err != nil {
		return errors.Wrap(err, "fetchField")
	}
	if len(ids) == 0 {
		return nil
	}
	return db.Purge(man, "id", ids, true)
}

func scanExpiredBillingResources(ctx context.Context) {
	billingResourceManagers := []IBillingModelManager{
		GuestManager,
		DBInstanceManager,
		ElasticcacheManager,
	}

	for _, manager := range billingResourceManagers {
		expiredModels, err := manager.GetExpiredModels(30)
		if err != nil {
			log.Errorf("unable to fetchExpiredModels: %v", err)
			continue
		}

		resourceIds := []string{}
		for _, model := range expiredModels {
			err = BillingResourceCheckManager.Create(ctx, model, manager.Keyword())
			if err != nil {
				log.Errorf("unable to create billing_resource_check for resource %s %s", manager.Keyword(), model.GetId())
				continue
			}
			resourceIds = append(resourceIds, model.GetId())
		}

		err = BillingResourceCheckManager.clean(manager.Keyword(), resourceIds)
		if err != nil {
			log.Errorf("unable to clean billing_resource_check for resource %s", manager.Keyword())
			continue
		}
	}
}

func CheckBillingResourceExpireAt(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	scanExpiredBillingResources(ctx)

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
		resources, err := BillingResourceCheckManager.GetExpiredModels(advanceDay)
		if err != nil {
			log.Errorf("unable to fetchExpiredModels: %s", err.Error())
			continue
		}

		for i := range resources {
			detailsDecro := func(ctx context.Context, details *jsonutils.JSONDict) {
				details.Set("advance_days", jsonutils.NewInt(int64(advanceDay)))
			}
			notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
				Obj:                 &resources[i],
				ObjDetailsDecorator: detailsDecro,
				Action:              notifyclient.ActionExpiredRelease,
				AdvanceDays:         advanceDay,
			})
		}
	}
}
