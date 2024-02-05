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

package usages

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/hashcache"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	historyUsageCache = hashcache.NewCache(1024, time.Second*300) // 5 minutes, 1024 buckets cache
)

type TimeRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Interval  string    `json:"interval"`
}

func (self *TimeRange) ValidateInput() error {
	if self.StartDate.IsZero() {
		return httperrors.NewMissingParameterError("start_date")
	}
	if self.EndDate.IsZero() {
		return httperrors.NewMissingParameterError("end_date")
	}
	if len(self.Interval) == 0 {
		return httperrors.NewMissingParameterError("interval")
	}
	if self.StartDate.After(self.EndDate) {
		return httperrors.NewInputParameterError("start_date should befor end_date")
	}
	switch self.Interval {
	case "hour":
		if self.EndDate.Sub(self.StartDate) > time.Hour*72 {
			return httperrors.NewOutOfRangeError("The time interval exceeds 72 hours")
		}
	case "day":
		if self.EndDate.Sub(self.StartDate) > time.Hour*24*31 {
			return httperrors.NewOutOfRangeError("The time interval exceeds 31 days")
		}
	case "month":
		if self.EndDate.Sub(self.StartDate) > time.Hour*24*365 {
			return httperrors.NewOutOfRangeError("The time interval exceeds 1 year")
		}
	case "year":
		if self.EndDate.Sub(self.StartDate) > time.Hour*24*365*20 {
			return httperrors.NewOutOfRangeError("The time interval exceeds 20 year")
		}
	default:
		return httperrors.NewInputParameterError("invalid interval %s", self.Interval)
	}
	return nil
}

func getHistoryCacheKey(
	scope rbacscope.TRbacScope,
	userCred mcclient.IIdentityProvider,
	timeRange *TimeRange,
	includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) string {
	type KeyStruct struct {
		Scope     rbacscope.TRbacScope `json:"scope"`
		Domain    string               `json:"domain"`
		Project   string               `json:"project"`
		System    bool                 `json:"system"`
		TimeRange *TimeRange           `json:"time_range"`

		PolicyResult rbacutils.SPolicyResult `json:"policy_result"`
	}
	key := KeyStruct{}
	key.Scope = scope
	switch scope {
	case rbacscope.ScopeSystem:
	case rbacscope.ScopeDomain:
		key.Domain = userCred.GetProjectDomainId()
	case rbacscope.ScopeProject:
		key.Project = userCred.GetProjectId()
	}
	key.TimeRange = timeRange
	key.System = includeSystem
	key.PolicyResult = policyResult
	jsonObj := jsonutils.Marshal(key)
	return jsonObj.QueryString()
}

func AddHistoryUsageHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/history-usages", prefix)
	for key, f := range map[string]appsrv.FilterHandler{
		"": historyRangeObjHandler(nil, ReportGeneralHistoryUsage),
	} {
		addHistoryHandler(prefix, key, f, app)
	}
}

func addHistoryHandler(prefix, rangeObjKey string, hf appsrv.FilterHandler, app *appsrv.Application) {
	ahf := auth.Authenticate(hf)
	name := "get_history_usage"
	if len(rangeObjKey) != 0 {
		prefix = fmt.Sprintf("%s/%ss/<id>", prefix, rangeObjKey)
		name = fmt.Sprintf("get_%s_history_usage", rangeObjKey)
	}
	app.AddHandler2("GET", prefix, ahf, nil, name, nil)
}

type objHistoryUsageFunc func(context.Context, mcclient.TokenCredential, rbacscope.TRbacScope, mcclient.IIdentityProvider, *TimeRange, bool, rbacutils.SPolicyResult) (Usage, error)

func historyRangeObjHandler(
	manager db.IStandaloneModelManager,
	reporter objHistoryUsageFunc,
) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
		ownerId, scope, err, result := db.FetchUsageOwnerScope(ctx, userCred, getQuery(r))
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		query := getQuery(r)
		tags := rbacutils.SPolicyResult{Result: rbacutils.Allow}
		query.Unmarshal(&tags)
		result = result.Merge(tags)
		log.Debugf("ownerId: %s scope: %s result: %s", ownerId, scope, result.String())
		timeRange := &TimeRange{}
		query.Unmarshal(timeRange)
		err = timeRange.ValidateInput()
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		includeSystem := jsonutils.QueryBoolean(query, "system", false)
		refresh := jsonutils.QueryBoolean(query, "refresh", false)
		key := getHistoryCacheKey(scope, ownerId, timeRange, includeSystem, result)
		if !refresh {
			cached := historyUsageCache.Get(key)
			if cached != nil {
				response(w, "history-usage", cached)
				return
			}
		}
		usage, err := reporter(ctx, userCred, scope, ownerId, timeRange, includeSystem, result)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		historyUsageCache.AtomicSet(key, usage)
		response(w, "history-usage", usage)
	}
}

func ReportGeneralHistoryUsage(
	ctx context.Context,
	userToken mcclient.TokenCredential,
	scope rbacscope.TRbacScope,
	userCred mcclient.IIdentityProvider,
	timeRange *TimeRange,
	includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) (count Usage, err error) {
	count = make(map[string]interface{})

	if scope == rbacscope.ScopeSystem {
		count = HistoryUsage(ctx, userToken, timeRange, rbacscope.ScopeSystem, userCred, includeSystem, policyResult)
	}

	if scope == rbacscope.ScopeDomain && len(userCred.GetProjectDomainId()) > 0 {
		count = HistoryUsage(ctx, userToken, timeRange, rbacscope.ScopeDomain, userCred, includeSystem, policyResult)
	}

	if scope == rbacscope.ScopeProject && len(userCred.GetProjectId()) > 0 {
		count = HistoryUsage(ctx, userToken, timeRange, rbacscope.ScopeProject, userCred, includeSystem, policyResult)
	}
	return
}

func HistoryUsage(ctx context.Context, userCred mcclient.TokenCredential, timeRange *TimeRange, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, includeSystem bool, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userCred, models.GuestManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	format := ""
	switch timeRange.Interval {
	case "hour":
		format = "%Y-%m-%d %h-00-00"
	case "day":
		format = "%Y-%m-%d"
	case "month":
		format = "%Y-%m"
	case "year":
		format = "%Y"
	default:
		return count
	}

	for _, manager := range []db.IModelManager{
		models.GuestManager,
		models.ElasticipManager,
		models.DBInstanceManager,
		models.LoadbalancerManager,
		models.HostManager,
		models.VpcManager,
		models.ElasticcacheManager,
		models.CloudaccountManager,
		models.DiskManager,
		models.DnsZoneManager,
		models.KubeClusterManager,
		models.NatGatewayManager,
		models.BucketManager,
		models.MongoDBManager,
	} {
		usage, _ := historyUsage(ctx, manager, scope, ownerId, timeRange, format, includeSystem, false, results)
		count[manager.Keyword()] = usage
	}

	usage, _ := historyUsage(ctx, models.HostManager, scope, ownerId, timeRange, format, includeSystem, true, results)
	count["baremetal"] = usage

	return count
}

type SHistoryUsage struct {
	Date  string
	Count int
}

func historyUsage(
	ctx context.Context,
	manager db.IModelManager,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	timeRange *TimeRange,
	format string,
	includeSystem bool,
	isBaremetal bool,
	policyResult rbacutils.SPolicyResult,
) ([]SHistoryUsage, error) {
	ret := []SHistoryUsage{}

	date, _start := []string{}, timeRange.StartDate
	for _start.Before(timeRange.EndDate) {
		switch timeRange.Interval {
		case "hour":
			date = append(date, _start.Format("2006-01-02 15:00:00"))
			_start = _start.Add(time.Hour)
		case "day":
			date = append(date, _start.Format("2006-01-02"))
			_start = _start.AddDate(0, 0, 1)
		case "month":
			date = append(date, _start.Format("2006-01"))
			_start = _start.AddDate(0, 1, 0)
		case "year":
			date = append(date, _start.Format("2006"))
			_start = _start.AddDate(1, 0, 0)
		}
	}
	dqs := []sqlchemy.IQuery{}
	for _, d := range date {
		dsq := manager.Query().SubQuery()
		dq := dsq.Query(
			sqlchemy.NewConstField(d).Label("date"),
		)
		dqs = append(dqs, dq)
	}

	dsq := sqlchemy.Union(dqs...).Query().SubQuery()

	gq := manager.Query()

	if manager.Keyword() == models.HostManager.Keyword() {
		if isBaremetal {
			gq = gq.Filter(sqlchemy.AND(
				sqlchemy.IsTrue(gq.Field("is_baremetal")),
				sqlchemy.Equals(gq.Field("host_type"), api.HOST_TYPE_BAREMETAL),
			))
		} else {
			gq = gq.Filter(sqlchemy.OR(
				sqlchemy.IsFalse(gq.Field("is_baremetal")),
				sqlchemy.NotEquals(gq.Field("host_type"), api.HOST_TYPE_BAREMETAL),
			))
		}
	}

	switch scope {
	case rbacscope.ScopeSystem:
	case rbacscope.ScopeDomain:
		gq = gq.Filter(sqlchemy.Equals(gq.Field("domain_id"), ownerId.GetProjectDomainId()))
	case rbacscope.ScopeProject:
		gq = gq.Filter(sqlchemy.Equals(gq.Field("tenant_id"), ownerId.GetProjectId()))
	}

	gq = db.ObjectIdQueryWithPolicyResult(ctx, gq, manager, policyResult)

	if _, ok := manager.(db.IVirtualModelManager); ok && !includeSystem {
		gq = gq.Filter(sqlchemy.OR(
			sqlchemy.IsNull(gq.Field("is_system")), sqlchemy.IsFalse(gq.Field("is_system"))))
	}

	hq := gq.Copy().LT("created_at", timeRange.StartDate)
	hcnt, err := hq.CountWithError()
	if err != nil {
		return nil, errors.Wrapf(err, "CountWithError")
	}
	gq = gq.GE("created_at", timeRange.StartDate).LE("created_at", timeRange.EndDate)

	mq := gq.SubQuery()

	sq := mq.Query(
		sqlchemy.DATE_FORMAT("date", mq.Field("created_at"), format),
		sqlchemy.COUNT("count", mq.Field("id")),
	).GroupBy("date")
	msq := sq.SubQuery()

	q := dsq.Query(
		dsq.Field("date"),
		msq.Field("count"),
	).LeftJoin(msq, sqlchemy.Equals(msq.Field("date"), dsq.Field("date"))).Asc(dsq.Field("date"))

	err = q.All(&ret)
	if err != nil {
		return nil, err
	}
	prev := hcnt
	for i := range ret {
		ret[i].Count = ret[i].Count + prev
		prev = ret[i].Count
	}
	return ret, err
}
