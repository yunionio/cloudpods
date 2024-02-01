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

	json "yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type Usage map[string]interface{}

func (u Usage) update(nu Usage) Usage {
	for k, v := range nu {
		u[k] = v
	}
	return u
}

func (u Usage) Add(key string, value interface{}) Usage {
	u[key] = value
	return u
}

func (u Usage) Get(key string) interface{} {
	return u[key]
}

func (u Usage) Include(nus ...Usage) Usage {
	for _, nu := range nus {
		u.update(nu)
	}
	return u
}

type objUsageFunc func(mcclient.TokenCredential, rbacscope.TRbacScope, mcclient.IIdentityProvider, bool, []db.IStandaloneModel, []string, []string, []string, string, bool, rbacutils.SPolicyResult) (Usage, error)

func getRangeObjId(ctx context.Context) (string, error) {
	params := appctx.AppContextParams(ctx)
	objId := params["<id>"]
	if len(objId) == 0 {
		return "", fmt.Errorf("Object %q id must specified", objId)
	}
	return objId, nil
}

func getRangeObj(ctx context.Context, man db.IStandaloneModelManager, userCred mcclient.TokenCredential) (db.IStandaloneModel, error) {
	if man == nil {
		return nil, nil
	}
	id, err := getRangeObjId(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getRangeObjId")
	}
	m, err := man.FetchByIdOrName(userCred, id)
	if err != nil {
		return nil, errors.Wrap(err, "man.FetchByIdOrName")
	}
	return m.(db.IStandaloneModel), nil
}

func rangeObjHandler(
	manager db.IStandaloneModelManager,
	reporter objUsageFunc,
) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
		obj, err := getRangeObj(ctx, manager, userCred)
		if err != nil {
			httperrors.NotFoundError(ctx, w, "%v", err)
			return
		}
		ownerId, scope, err, result := db.FetchUsageOwnerScope(ctx, userCred, getQuery(r))
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		query := getQuery(r)
		tags := rbacutils.SPolicyResult{Result: rbacutils.Allow}
		query.Unmarshal(&tags)
		result = result.Merge(tags)
		isOwner := false
		if scope == rbacscope.ScopeDomain && obj != nil && db.IsObjectRbacAllowed(ctx, obj, userCred, policy.PolicyActionGet, "usage") == nil {
			isOwner = true
		}
		log.Debugf("ownerId: %s isOwner: %v scope: %s result: %s", ownerId, isOwner, scope, result.String())
		hostTypes := json.GetQueryStringArray(query, "host_type")
		// resourceTypes := json.GetQueryStringArray(query, "resource_type")
		providers := json.GetQueryStringArray(query, "provider")
		brands := json.GetQueryStringArray(query, "brand")
		cloudEnv, _ := query.GetString("cloud_env")
		includeSystem := json.QueryBoolean(query, "system", false)
		var rangeObjs []db.IStandaloneModel
		if obj != nil {
			rangeObjs = []db.IStandaloneModel{obj}
		}
		refresh := json.QueryBoolean(query, "refresh", false)
		key := getCacheKey(scope, ownerId, isOwner, rangeObjs, hostTypes, providers, brands, cloudEnv, includeSystem, result)
		if !refresh {
			cached := usageCache.Get(key)
			if cached != nil {
				response(w, "usage", cached)
				return
			}
		}
		usage, err := reporter(userCred, scope, ownerId, isOwner, rangeObjs, hostTypes, providers, brands, cloudEnv, includeSystem, result)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		usageCache.AtomicSet(key, usage)
		response(w, "usage", usage)
	}
}

func addHandler(prefix, rangeObjKey string, hf appsrv.FilterHandler, app *appsrv.Application) {
	ahf := auth.Authenticate(hf)
	name := "get_usage"
	if len(rangeObjKey) != 0 {
		prefix = fmt.Sprintf("%s/%ss/<id>", prefix, rangeObjKey)
		name = fmt.Sprintf("get_%s_usage", rangeObjKey)
	}
	app.AddHandler2("GET", prefix, ahf, nil, name, nil)
}

func AddUsageHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/usages", prefix)
	for key, f := range map[string]appsrv.FilterHandler{
		"":              rangeObjHandler(nil, ReportGeneralUsage),
		"zone":          rangeObjHandler(models.ZoneManager, ReportZoneUsage),
		"wire":          rangeObjHandler(models.WireManager, ReportWireUsage),
		"schedtag":      rangeObjHandler(models.SchedtagManager, ReportSchedtagUsage),
		"host":          rangeObjHandler(models.HostManager, ReportHostUsage),
		"cloudaccount":  rangeObjHandler(models.CloudaccountManager, ReportCloudAccountUsage),
		"cloudprovider": rangeObjHandler(models.CloudproviderManager, ReportCloudProviderUsage),
		"cloudregion":   rangeObjHandler(models.CloudregionManager, ReportCloudRegionUsage),
	} {
		addHandler(prefix, key, f, app)
	}
}

func response(w http.ResponseWriter, key string, obj interface{}) {
	body := map[string]interface{}{
		key: obj,
	}
	appsrv.SendStruct(w, body)
}

func getQuery(r *http.Request) json.JSONObject {
	query, e := json.ParseQueryString(r.URL.RawQuery)
	if e != nil {
		log.Errorf("Parse query string %q: %v", r.URL.RawQuery, e)
	}
	return query
}

func ReportHostUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, hosts []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(userToken, scope, userCred, isOwner, hosts, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportWireUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, wires []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(userToken, scope, userCred, isOwner, wires, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportCloudAccountUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, accounts []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(userToken, scope, userCred, isOwner, accounts, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportCloudProviderUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, managers []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(userToken, scope, userCred, isOwner, managers, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportSchedtagUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, schedtags []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(userToken, scope, userCred, isOwner, schedtags, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportZoneUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, zones []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(userToken, scope, userCred, isOwner, zones, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportCloudRegionUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, cloudRegions []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(userToken, scope, userCred, isOwner, cloudRegions, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func getSystemGeneralUsage(
	userToken mcclient.TokenCredential,
	userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostTypes []string,
	providers []string, brands []string, cloudEnv string, includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) (Usage, error) {
	count := RegionUsage(rangeObjs, providers, brands, cloudEnv)
	zone := ZoneUsage(rangeObjs, providers, brands, cloudEnv)
	count.Include(zone)

	var pmemTotal float64
	var pcpuTotal float64

	hostEnabledUsage := HostEnabledUsage(userToken, "", userCred, rbacscope.ScopeSystem, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult)
	if !gotypes.IsNil(hostEnabledUsage.Get("enabled_hosts.memory")) {
		pmemTotal = float64(hostEnabledUsage.Get("enabled_hosts.memory").(int64))
	}
	if !gotypes.IsNil(hostEnabledUsage.Get("enabled_hosts.cpu")) {
		pcpuTotal = float64(hostEnabledUsage.Get("enabled_hosts.cpu").(int64))
	}
	if len(rangeObjs) > 0 && rangeObjs[0].Keyword() == "host" {
		host := rangeObjs[0].(*models.SHost)
		pmemTotal = float64(host.MemSize)
		pcpuTotal = float64(host.CpuCount)
		count.Add("memory", host.MemSize)
		count.Add("cpu", host.CpuCount)
		count.Add("memory.virtual", int64(host.GetVirtualMemorySize()))
		count.Add("cpu.virtual", int64(host.GetVirtualCPUCount()))
	}

	guestRunningUsage := GuestRunningUsage(userToken, "all.running_servers", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, policyResult)
	var runningMem int
	var runningCpu int
	if !gotypes.IsNil(guestRunningUsage.Get("all.running_servers.memory")) {
		runningMem = guestRunningUsage.Get("all.running_servers.memory").(int)
	}
	if !gotypes.IsNil(guestRunningUsage.Get("all.running_servers.cpu")) {
		runningCpu = guestRunningUsage.Get("all.running_servers.cpu").(int)
	}

	// containerRunningUsage := containerUsage("all.containers", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv)
	// containerRunningMem := containerRunningUsage.Get("all.containers.memory").(int)
	// containerRunningCpu := containerRunningUsage.Get("all.containers.cpu").(int)
	// runningMem += containerRunningMem
	// runningCpu += containerRunningCpu
	runningCpuCmtRate := 0.0
	runningMemCmtRate := 0.0
	if pmemTotal > 0 {
		runningMemCmtRate = utils.FloatRound(float64(runningMem)/pmemTotal, 2)
	}
	if pcpuTotal > 0 {
		runningCpuCmtRate = utils.FloatRound(float64(runningCpu)/pcpuTotal, 2)
	}
	count.Add("all.memory_commit_rate.running", runningMemCmtRate)
	count.Add("all.cpu_commit_rate.running", runningCpuCmtRate)

	lastWeek := time.Now().Add(-7 * 24 * time.Hour)
	lastMonth := time.Now().Add(-30 * 24 * time.Hour)
	count.Include(
		VpcUsage(userToken, "all", providers, brands, cloudEnv, nil, rbacscope.ScopeSystem, rangeObjs, policyResult),

		DnsZoneUsage(userToken, "", nil, rbacscope.ScopeSystem, policyResult),

		HostAllUsage(userToken, "", userCred, rbacscope.ScopeSystem, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult),
		// HostAllUsage("prepaid_pool", userCred, rbacscope.ScopeSystem, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// HostAllUsage("any_pool", userCred, rbacscope.ScopeSystem, rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		hostEnabledUsage,
		// HostEnabledUsage("prepaid_pool", userCred, rbacscope.ScopeSystem, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// HostEnabledUsage("any_pool", userCred, rbacscope.ScopeSystem, rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		BaremetalUsage(userToken, userCred, rbacscope.ScopeSystem, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),

		StorageUsage(userToken, "", rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, includeSystem, rbacscope.ScopeSystem, nil, policyResult),
		StorageUsage(userToken, "system", rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, true, rbacscope.ScopeSystem, nil, policyResult),
		// StorageUsage("prepaid_pool", rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false, includeSystem, rbacscope.ScopeSystem, nil),
		// StorageUsage("any_pool", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false, includeSystem, rbacscope.ScopeSystem, nil),
		// StorageUsage("any_pool.system", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false, true, rbacscope.ScopeSystem, nil),
		// StorageUsage("any_pool.pending_delete", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, true, includeSystem, rbacscope.ScopeSystem, nil),
		// StorageUsage("any_pool.pending_delete.system", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, true, true, rbacscope.ScopeSystem, nil),

		GuestNormalUsage(userToken, "all.servers", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, nil, policyResult),
		GuestNormalUsage(userToken, "all.servers.last_week", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, &lastWeek, policyResult),
		GuestNormalUsage(userToken, "all.servers.last_month", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, &lastMonth, policyResult),
		// GuestNormalUsage("all.servers.prepaid_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestNormalUsage("all.servers.any_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),

		GuestPendingDeleteUsage(userToken, "all.pending_delete_servers", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, nil, policyResult),
		GuestPendingDeleteUsage(userToken, "all.pending_delete_servers.last_week", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, &lastWeek, policyResult),
		GuestPendingDeleteUsage(userToken, "all.pending_delete_servers.last_month", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, &lastMonth, policyResult),
		// GuestPendingDeleteUsage("all.pending_delete_servers.prepaid_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestNormalUsage("all.servers.prepaid_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestNormalUsage("all.servers.any_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),

		// GuestPendingDeleteUsage("all.pending_delete_servers", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem),
		// GuestPendingDeleteUsage("all.pending_delete_servers.prepaid_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestPendingDeleteUsage("all.pending_delete_servers.any_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),

		GuestReadyUsage(userToken, "all.ready_servers", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, policyResult),
		// GuestReadyUsage("all.ready_servers.prepaid_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestReadyUsage("all.ready_servers.any_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),
		// GuestRunningUsage("all.running_servers.prepaid_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestRunningUsage("all.running_servers.any_pool", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),

		guestRunningUsage,
		// containerRunningUsage,

		IsolatedDeviceUsage(userToken, "", rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult),
		// IsolatedDeviceUsage("prepaid_pool", rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// IsolatedDeviceUsage("any_pool", rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		WireUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),
		NetworkUsage(userToken, "all", rbacscope.ScopeSystem, nil, providers, brands, cloudEnv, rangeObjs, policyResult),

		EipUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		BucketUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		SnapshotUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		InstanceSnapshotUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		LoadbalancerUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		DBInstanceUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		MongoDBUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticSearchUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		KafkaUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticCacheUsage(userToken, rbacscope.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),
	)

	return count, nil
}

func getDomainGeneralUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, cred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) (Usage, error) {
	lastWeek := time.Now().Add(-7 * 24 * time.Hour)
	lastMonth := time.Now().Add(-30 * 24 * time.Hour)
	count := GuestNormalUsage(userToken, getKey(scope, "servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, nil, policyResult)

	var pmemTotal float64
	var pcpuTotal float64

	hostEnabledUsage := HostEnabledUsage(userToken, "", cred, rbacscope.ScopeDomain, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult)
	if !gotypes.IsNil(hostEnabledUsage.Get("domain.enabled_hosts.memory")) {
		pmemTotal = float64(hostEnabledUsage.Get("domain.enabled_hosts.memory").(int64))
	}
	if !gotypes.IsNil(hostEnabledUsage.Get("domain.enabled_hosts.cpu")) {
		pcpuTotal = float64(hostEnabledUsage.Get("domain.enabled_hosts.cpu").(int64))
	}

	guestRunningUsage := GuestRunningUsage(userToken, "domain.running_servers", rbacscope.ScopeDomain, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, policyResult)
	var runningMem int
	var runningCpu int
	if !gotypes.IsNil(guestRunningUsage.Get("domain.running_servers.memory")) {
		runningMem = guestRunningUsage.Get("domain.running_servers.memory").(int)
	}
	if !gotypes.IsNil(guestRunningUsage.Get("domain.running_servers.cpu")) {
		runningCpu = guestRunningUsage.Get("domain.running_servers.cpu").(int)
	}

	runningCpuCmtRate := 0.0
	runningMemCmtRate := 0.0
	if pmemTotal > 0 {
		runningMemCmtRate = utils.FloatRound(float64(runningMem)/pmemTotal, 2)
	}
	if pcpuTotal > 0 {
		runningCpuCmtRate = utils.FloatRound(float64(runningCpu)/pcpuTotal, 2)
	}
	count.Add("domain.memory_commit_rate.running", runningMemCmtRate)
	count.Add("domain.cpu_commit_rate.running", runningCpuCmtRate)

	count.Include(
		VpcUsage(userToken, "domain", providers, brands, cloudEnv, cred, rbacscope.ScopeDomain, rangeObjs, policyResult),

		DnsZoneUsage(userToken, "domain", cred, rbacscope.ScopeDomain, policyResult),

		HostAllUsage(userToken, "", cred, rbacscope.ScopeDomain, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult),
		// HostAllUsage("prepaid_pool", cred, rbacscope.ScopeDomain, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// HostAllUsage("any_pool", cred, rbacscope.ScopeDomain, rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		hostEnabledUsage,
		// HostEnabledUsage("prepaid_pool", cred, rbacscope.ScopeDomain, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// HostEnabledUsage("any_pool", cred, rbacscope.ScopeDomain, rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		BaremetalUsage(userToken, cred, rbacscope.ScopeDomain, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),

		StorageUsage(userToken, "", rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, false, rbacscope.ScopeDomain, cred, policyResult),
		StorageUsage(userToken, "system", rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, true, rbacscope.ScopeDomain, cred, policyResult),
		// StorageUsage("prepaid_pool", rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false, false, rbacscope.ScopeDomain, cred),
		// StorageUsage("any_pool", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false, false, rbacscope.ScopeDomain, cred),
		// StorageUsage("any_pool.system", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false, true, rbacscope.ScopeDomain, cred),
		// StorageUsage("any_pool.pending_delete", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, true, false, rbacscope.ScopeDomain, cred),
		// StorageUsage("any_pool.pending_delete.system", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, true, true, rbacscope.ScopeDomain, cred),

		GuestNormalUsage(userToken, getKey(scope, "servers.last_week"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastWeek, policyResult),
		GuestNormalUsage(userToken, getKey(scope, "servers.last_month"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastMonth, policyResult),
		// GuestNormalUsage(getKey(scope, "servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestNormalUsage(getKey(scope, "servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		guestRunningUsage,
		// GuestRunningUsage(getKey(scope, "running_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		// GuestRunningUsage(getKey(scope, "running_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestRunningUsage(getKey(scope, "running_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		GuestPendingDeleteUsage(userToken, getKey(scope, "pending_delete_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, nil, policyResult),
		GuestPendingDeleteUsage(userToken, getKey(scope, "pending_delete_servers.last_week"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastWeek, policyResult),
		GuestPendingDeleteUsage(userToken, getKey(scope, "pending_delete_servers.last_month"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastMonth, policyResult),
		// GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		GuestReadyUsage(userToken, getKey(scope, "ready_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, policyResult),
		// GuestReadyUsage(getKey(scope, "ready_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestReadyUsage(getKey(scope, "ready_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		WireUsage(userToken, scope, cred, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),
		NetworkUsage(userToken, getKey(scope, ""), scope, cred, providers, brands, cloudEnv, rangeObjs, policyResult),

		IsolatedDeviceUsage(userToken, "", scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult),

		EipUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		BucketUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		// nicsUsage("domain", rangeObjs, hostTypes, providers, brands, cloudEnv, scope, cred),

		SnapshotUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		InstanceSnapshotUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		LoadbalancerUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		DBInstanceUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		MongoDBUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticSearchUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		KafkaUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticCacheUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),
	)
	return count, nil
}

func getProjectGeneralUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, cred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) (Usage, error) {
	lastWeek := time.Now().Add(-7 * 24 * time.Hour)
	lastMonth := time.Now().Add(-30 * 24 * time.Hour)
	count := GuestNormalUsage(userToken, getKey(scope, "servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, nil, policyResult)

	count.Include(
		GuestNormalUsage(userToken, getKey(scope, "servers.last_week"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastWeek, policyResult),
		GuestNormalUsage(userToken, getKey(scope, "servers.last_month"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastMonth, policyResult),
		// GuestNormalUsage(getKey(scope, "servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestNormalUsage(getKey(scope, "servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),
		GuestRunningUsage(userToken, getKey(scope, "running_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, policyResult),
		// GuestRunningUsage(getKey(scope, "running_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestRunningUsage(getKey(scope, "running_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		GuestPendingDeleteUsage(userToken, getKey(scope, "pending_delete_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, nil, policyResult),
		GuestPendingDeleteUsage(userToken, getKey(scope, "pending_delete_servers.last_week"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastWeek, policyResult),
		GuestPendingDeleteUsage(userToken, getKey(scope, "pending_delete_servers.last_month"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastMonth, policyResult),
		// GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		GuestReadyUsage(userToken, getKey(scope, "ready_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, policyResult),
		// GuestReadyUsage(getKey(scope, "ready_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestReadyUsage(getKey(scope, "ready_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		WireUsage(userToken, scope, cred, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),
		NetworkUsage(userToken, getKey(scope, ""), scope, cred, providers, brands, cloudEnv, rangeObjs, policyResult),

		EipUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		BucketUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		DisksUsage(userToken, getKey(scope, "disks"), rangeObjs, hostTypes, nil, providers, brands, cloudEnv, scope, cred, false, false, policyResult),
		DisksUsage(userToken, getKey(scope, "disks.system"), rangeObjs, hostTypes, nil, providers, brands, cloudEnv, scope, cred, false, true, policyResult),
		DisksUsage(userToken, getKey(scope, "pending_delete_disks"), rangeObjs, hostTypes, nil, providers, brands, cloudEnv, scope, cred, true, false, policyResult),
		DisksUsage(userToken, getKey(scope, "pending_delete_disks.system"), rangeObjs, hostTypes, nil, providers, brands, cloudEnv, scope, cred, true, true, policyResult),

		// nicsUsage("", rangeObjs, hostTypes, providers, brands, cloudEnv, scope, cred),

		SnapshotUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		InstanceSnapshotUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		LoadbalancerUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		DBInstanceUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		MongoDBUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticSearchUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		KafkaUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticCacheUsage(userToken, scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),
	)

	return count, nil
}

func ReportGeneralUsage(
	userToken mcclient.TokenCredential,
	scope rbacscope.TRbacScope,
	userCred mcclient.IIdentityProvider,
	isOwner bool,
	rangeObjs []db.IStandaloneModel,
	hostTypes []string,
	providers []string,
	brands []string,
	cloudEnv string,
	includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) (count Usage, err error) {
	count = make(map[string]interface{})

	// if scope == rbacscope.ScopeSystem || isOwner {
	if scope == rbacscope.ScopeSystem {
		count, err = getSystemGeneralUsage(userToken, userCred, rangeObjs, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
		if err != nil {
			return
		}
	}

	// if scope.HigherEqual(rbacscope.ScopeDomain) && len(userCred.GetProjectDomainId()) > 0 {
	if scope == rbacscope.ScopeDomain && len(userCred.GetProjectDomainId()) > 0 {
		commonUsage, err := getDomainGeneralUsage(userToken, rbacscope.ScopeDomain, userCred, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult)
		if err == nil {
			count.Include(commonUsage)
		}
	}

	// if scope.HigherEqual(rbacscope.ScopeProject) && len(userCred.GetProjectId()) > 0 {
	if scope == rbacscope.ScopeProject && len(userCred.GetProjectId()) > 0 {
		commonUsage, err := getProjectGeneralUsage(userToken, rbacscope.ScopeProject, userCred, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult)
		if err == nil {
			count.Include(commonUsage)
		}
	}
	return
}

func RegionUsage(rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string) Usage {
	q := models.CloudregionManager.Query()

	if len(rangeObjs) > 0 || len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		subq := models.VpcManager.Query("cloudregion_id")
		subq = models.CloudProviderFilter(subq, subq.Field("manager_id"), providers, brands, cloudEnv)
		subq = models.RangeObjectsFilter(subq, rangeObjs, nil, nil, subq.Field("manager_id"), nil, nil)
		q = q.In("id", subq.SubQuery())
	}

	count := make(map[string]interface{})
	count["regions"], _ = q.CountWithError()
	return count
}

func ZoneUsage(rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string) Usage {
	q := models.ZoneManager.Query()

	if len(rangeObjs) > 0 || len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		subq := models.HostManager.Query("zone_id")
		subq = models.CloudProviderFilter(subq, subq.Field("manager_id"), providers, brands, cloudEnv)
		subq = models.RangeObjectsFilter(subq, rangeObjs, nil, nil, subq.Field("manager_id"), nil, nil)
		q = q.In("id", subq.SubQuery())
	}

	count := make(map[string]interface{})
	count["zones"], _ = q.CountWithError()
	return count
}

func VpcUsage(userToken mcclient.TokenCredential, prefix string, providers []string, brands []string, cloudEnv string, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, rangeObjs []db.IStandaloneModel, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})

	results := db.UsagePolicyCheck(userToken, models.VpcManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		return count
	}

	q := models.VpcManager.Query().IsFalse("is_emulated")
	if len(rangeObjs) > 0 || len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		q = models.CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
		q = models.RangeObjectsFilter(q, rangeObjs, nil, nil, q.Field("manager_id"), nil, nil)
	}
	if scope == rbacscope.ScopeDomain {
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}

	q = db.ObjectIdQueryWithPolicyResult(q, models.VpcManager, results)

	key := "vpcs"
	if len(prefix) > 0 {
		key = fmt.Sprintf("%s.vpcs", prefix)
	}
	count[key], _ = q.CountWithError()
	return count
}

func DnsZoneUsage(userToken mcclient.TokenCredential, prefix string, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})

	results := db.UsagePolicyCheck(userToken, models.DnsZoneManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		return count
	}

	q := models.DnsZoneManager.Query()
	if scope == rbacscope.ScopeDomain {
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}

	q = db.ObjectIdQueryWithPolicyResult(q, models.DnsZoneManager, results)

	key := "dns_zones"
	if len(prefix) > 0 {
		key = fmt.Sprintf("%s.dns_zones", prefix)
	}
	count[key], _ = q.CountWithError()
	return count
}

func StorageUsage(
	userToken mcclient.TokenCredential,
	prefix string,
	rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	pendingDeleted bool, includeSystem bool,
	scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider,
	policyResult rbacutils.SPolicyResult,
) Usage {
	results := db.UsagePolicyCheck(userToken, models.StorageManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		return map[string]interface{}{}
	}
	sPrefix := getSysKey(scope, "storages")
	dPrefix := getKey(scope, "disks")
	if len(prefix) > 0 {
		sPrefix = fmt.Sprintf("%s.%s", sPrefix, prefix)
		dPrefix = fmt.Sprintf("%s.%s", dPrefix, prefix)
	}
	count := make(map[string]interface{})
	result := models.StorageManager.TotalCapacity(
		rangeObjs,
		hostTypes, resourceTypes,
		providers, brands, cloudEnv,
		scope, ownerId,
		pendingDeleted, includeSystem,
		true,
		results,
	)
	count[sPrefix] = result.Capacity
	for s, capa := range result.StorageTypeCapacity {
		count[fmt.Sprintf("%s.storage_type.%s", sPrefix, s)] = capa
	}
	for m, capa := range result.MediumeCapacity {
		count[fmt.Sprintf("%s.medium_type.%s", sPrefix, m)] = capa
	}
	count[fmt.Sprintf("%s.virtual", sPrefix)] = result.CapacityVirtual
	count[fmt.Sprintf("%s.owner", dPrefix)] = result.CapacityUsed
	count[fmt.Sprintf("%s.count.owner", dPrefix)] = result.CountUsed
	count[fmt.Sprintf("%s.unready.owner", dPrefix)] = result.CapacityUnready
	count[fmt.Sprintf("%s.unready.count.owner", dPrefix)] = result.CountUnready
	count[fmt.Sprintf("%s.attached.owner", dPrefix)] = result.AttachedCapacity
	for s, capa := range result.AttachedStorageTypeCapacity {
		count[fmt.Sprintf("%s.attached.owner.storage_type.%s", dPrefix, s)] = capa
	}
	for m, capa := range result.AttachedMediumeCapacity {
		count[fmt.Sprintf("%s.attached.owner.medium_type.%s", dPrefix, m)] = capa
	}
	count[fmt.Sprintf("%s.attached.count.owner", dPrefix)] = result.CountAttached
	count[fmt.Sprintf("%s.detached.owner", dPrefix)] = result.DetachedCapacity
	for s, capa := range result.DetachedStorageTypeCapacity {
		count[fmt.Sprintf("%s.detached.owner.storage_type.%s", dPrefix, s)] = capa
	}
	for m, capa := range result.DetachedMediumeCapacity {
		count[fmt.Sprintf("%s.detached.owner.medium_type.%s", dPrefix, m)] = capa
	}
	count[fmt.Sprintf("%s.detached.count.owner", dPrefix)] = result.CountDetached
	count[fmt.Sprintf("%s.mounted.owner", dPrefix)] = result.AttachedCapacity
	count[fmt.Sprintf("%s.mounted.count.owner", dPrefix)] = result.CountAttached
	count[fmt.Sprintf("%s.unmounted.owner", dPrefix)] = result.DetachedCapacity
	count[fmt.Sprintf("%s.unmounted.count.owner", dPrefix)] = result.CountDetached

	storageCmtRate := 0.0
	if result.Capacity > 0 {
		storageCmtRate = utils.FloatRound(float64(result.CapacityUsed)/float64(result.Capacity), 2)
	}
	count[fmt.Sprintf("%s.commit_rate", sPrefix)] = storageCmtRate

	result = models.StorageManager.TotalCapacity(
		rangeObjs,
		hostTypes, resourceTypes,
		providers, brands, cloudEnv,
		scope, ownerId,
		pendingDeleted, includeSystem,
		false,
		policyResult,
	)

	count[fmt.Sprintf("%s", dPrefix)] = result.CapacityUsed
	for s, capa := range result.StorageTypeCapacityUsed {
		count[fmt.Sprintf("%s.storage_type.%s", dPrefix, s)] = capa
	}
	for m, capa := range result.MediumeCapacityUsed {
		count[fmt.Sprintf("%s.medium_type.%s", dPrefix, m)] = capa
	}

	count[fmt.Sprintf("%s.count", dPrefix)] = result.CountUsed
	count[fmt.Sprintf("%s.unready", dPrefix)] = result.CapacityUnready
	count[fmt.Sprintf("%s.unready.count", dPrefix)] = result.CountUnready
	count[fmt.Sprintf("%s.attached", dPrefix)] = result.AttachedCapacity
	for s, capa := range result.AttachedStorageTypeCapacity {
		count[fmt.Sprintf("%s.attached.storage_type.%s", dPrefix, s)] = capa
	}
	for m, capa := range result.AttachedMediumeCapacity {
		count[fmt.Sprintf("%s.attached.medium_type.%s", dPrefix, m)] = capa
	}
	count[fmt.Sprintf("%s.attached.count", dPrefix)] = result.CountAttached
	count[fmt.Sprintf("%s.detached", dPrefix)] = result.DetachedCapacity
	for s, capa := range result.DetachedStorageTypeCapacity {
		count[fmt.Sprintf("%s.detached.storage_type.%s", dPrefix, s)] = capa
	}
	for m, capa := range result.DetachedMediumeCapacity {
		count[fmt.Sprintf("%s.detached.medium_type.%s", dPrefix, m)] = capa
	}
	count[fmt.Sprintf("%s.detached.count", dPrefix)] = result.CountDetached
	count[fmt.Sprintf("%s.mounted", dPrefix)] = result.AttachedCapacity
	count[fmt.Sprintf("%s.mounted.count", dPrefix)] = result.CountAttached
	count[fmt.Sprintf("%s.unmounted", dPrefix)] = result.DetachedCapacity
	count[fmt.Sprintf("%s.unmounted.count", dPrefix)] = result.CountDetached

	return count
}

func DisksUsage(
	userToken mcclient.TokenCredential,
	dPrefix string,
	rangeObjs []db.IStandaloneModel,
	hostTypes []string,
	resourceTypes []string,
	providers []string,
	brands []string,
	cloudEnv string,
	scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider,
	pendingDeleted bool, includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) Usage {
	count := make(map[string]interface{})

	results := db.UsagePolicyCheck(userToken, models.StorageManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		return count
	}

	result := models.StorageManager.TotalCapacity(rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, scope, ownerId, pendingDeleted, includeSystem, false, results)
	count[dPrefix] = result.CapacityUsed
	count[fmt.Sprintf("%s.storage", dPrefix)] = result.Capacity
	count[fmt.Sprintf("%s.storage.virtual", dPrefix)] = result.CapacityVirtual
	count[fmt.Sprintf("%s.count", dPrefix)] = result.CountUsed
	count[fmt.Sprintf("%s.unready", dPrefix)] = result.CapacityUnready
	count[fmt.Sprintf("%s.unready.count", dPrefix)] = result.CountUnready
	count[fmt.Sprintf("%s.attached", dPrefix)] = result.AttachedCapacity
	count[fmt.Sprintf("%s.attached.count", dPrefix)] = result.CountAttached
	count[fmt.Sprintf("%s.detached", dPrefix)] = result.DetachedCapacity
	count[fmt.Sprintf("%s.detached.count", dPrefix)] = result.CountDetached
	count[fmt.Sprintf("%s.mounted", dPrefix)] = result.AttachedCapacity
	count[fmt.Sprintf("%s.mounted.count", dPrefix)] = result.CountAttached
	count[fmt.Sprintf("%s.unmounted", dPrefix)] = result.DetachedCapacity
	count[fmt.Sprintf("%s.unmounted.count", dPrefix)] = result.CountDetached
	return count
}

func WireUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})

	results := db.UsagePolicyCheck(userToken, models.WireManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		return count
	}

	result := models.WireManager.TotalCount(rangeObjs, hostTypes, providers, brands, cloudEnv, scope, userCred, results)
	count[getKey(scope, "wires")] = result.WiresCount - result.EmulatedWiresCount
	count[getKey(scope, "networks")] = result.NetCount
	// include nics for pending_deleted guests
	count[getKey(scope, "nics.guest")] = result.GuestNicCount
	// nics for pending_deleted guests
	count[getKey(scope, "nics.guest.pending_delete")] = result.PendingDeletedGuestNicCount
	count[getKey(scope, "nics.host")] = result.HostNicCount
	count[getKey(scope, "nics.reserve")] = result.ReservedCount
	count[getKey(scope, "nics.group")] = result.GroupNicCount
	count[getKey(scope, "nics.lb")] = result.LbNicCount
	count[getKey(scope, "nics.eip")] = result.EipNicCount
	count[getKey(scope, "nics.netif")] = result.NetifNicCount
	count[getKey(scope, "nics.db")] = result.DbNicCount
	count[getKey(scope, "nics")] = result.NicCount()

	return count
}

/*func nicsUsage(prefix string, rangeObjs []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider) Usage {
	count := make(map[string]interface{})
	result := models.WireManager.TotalCount(rangeObjs, hostTypes, providers, brands, cloudEnv, scope, ownerId)
	// including nics for pending_deleted guests
	count[prefixKey(prefix, "nics.guest")] = result.GuestNicCount
	// #nics for pending_deleted guests
	count[prefixKey(prefix, "nics.guest.pending_delete")] = result.PendingDeletedGuestNicCount
	count[prefixKey(prefix, "nics.group")] = result.GroupNicCount
	count[prefixKey(prefix, "nics.lb")] = result.LbNicCount
	count[prefixKey(prefix, "nics.db")] = result.DbNicCount
	count[prefixKey(prefix, "nics.eip")] = result.EipNicCount
	count[prefixKey(prefix, "nics")] = result.GuestNicCount + result.GroupNicCount + result.LbNicCount + result.DbNicCount + result.EipNicCount
	return count
}*/

func prefixKey(prefix string, key string) string {
	if len(prefix) > 0 {
		return prefix + "." + key
	} else {
		return key
	}
}

func NetworkUsage(userToken mcclient.TokenCredential, prefix string, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, providers []string, brands []string, cloudEnv string, rangeObjs []db.IStandaloneModel, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})

	results := db.UsagePolicyCheck(userToken, models.NetworkManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		return count
	}

	ret := models.NetworkManager.TotalPortCount(scope, userCred, providers, brands, cloudEnv, rangeObjs, results)
	for k, v := range ret {
		if len(k) > 0 {
			count[prefixKey(prefix, fmt.Sprintf("ports.%s", k))] = v.Count
			count[prefixKey(prefix, fmt.Sprintf("ports_exit.%s", k))] = v.CountExt
		} else {
			count[prefixKey(prefix, "ports")] = v.Count
			count[prefixKey(prefix, "ports_exit")] = v.CountExt
		}
	}
	return count
}

func HostAllUsage(userToken mcclient.TokenCredential, pref string, userCred mcclient.IIdentityProvider, scope rbacscope.TRbacScope, rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	prefix := getSysKey(scope, "hosts")
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	return hostUsage(userToken, userCred, scope, prefix, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, tristate.None, tristate.False, policyResult)
}

func HostEnabledUsage(userToken mcclient.TokenCredential, pref string, userCred mcclient.IIdentityProvider, scope rbacscope.TRbacScope, rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	prefix := getSysKey(scope, "enabled_hosts")
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	return hostUsage(userToken, userCred, scope, prefix, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, tristate.True, tristate.False, policyResult)
}

func BaremetalUsage(userToken mcclient.TokenCredential, userCred mcclient.IIdentityProvider, scope rbacscope.TRbacScope, rangeObjs []db.IStandaloneModel,
	hostTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	prefix := getSysKey(scope, "baremetals")
	count := hostUsage(userToken, userCred, scope, prefix, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, tristate.None, tristate.True, policyResult)
	delete(count, fmt.Sprintf("%s.memory.virtual", prefix))
	delete(count, fmt.Sprintf("%s.cpu.virtual", prefix))
	return count
}

func hostUsage(
	userToken mcclient.TokenCredential,
	userCred mcclient.IIdentityProvider, scope rbacscope.TRbacScope, prefix string,
	rangeObjs []db.IStandaloneModel, hostTypes []string,
	resourceTypes []string, providers []string, brands []string, cloudEnv string,
	enabled, isBaremetal tristate.TriState,
	policyResult rbacutils.SPolicyResult,
) Usage {
	count := make(map[string]interface{})

	results := db.UsagePolicyCheck(userToken, models.HostManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		return count
	}

	result := models.HostManager.TotalCount(userCred, scope, rangeObjs, "", "", hostTypes, resourceTypes, providers, brands, cloudEnv, enabled, isBaremetal, results)
	count[prefix] = result.Count
	count[fmt.Sprintf("%s.any_pool", prefix)] = result.Count
	count[fmt.Sprintf("%s.memory", prefix)] = result.Memory
	count[fmt.Sprintf("%s.memory.total", prefix)] = result.MemoryTotal
	count[fmt.Sprintf("%s.cpu", prefix)] = result.CPU
	count[fmt.Sprintf("%s.cpu.total", prefix)] = result.CPUTotal
	count[fmt.Sprintf("%s.memory.virtual", prefix)] = int64(result.MemoryVirtual)
	count[fmt.Sprintf("%s.cpu.virtual", prefix)] = int64(result.CPUVirtual)
	count[fmt.Sprintf("%s.memory.reserved", prefix)] = result.MemoryReserved
	count[fmt.Sprintf("%s.memory.reserved.isolated", prefix)] = result.IsolatedReservedMemory
	count[fmt.Sprintf("%s.cpu.reserved.isolated", prefix)] = result.IsolatedReservedCpu
	count[fmt.Sprintf("%s.storage.reserved.isolated", prefix)] = result.IsolatedReservedStorage
	count[fmt.Sprintf("%s.storage_gb", prefix)] = result.StorageSize / 1024
	return count
}

func GuestNormalUsage(userToken mcclient.TokenCredential, prefix string, scope rbacscope.TRbacScope, cred mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string,
	brands []string, cloudEnv string, includeSystem bool, since *time.Time, policyResult rbacutils.SPolicyResult) Usage {
	return guestUsage(userToken, prefix, scope, cred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, nil, false, includeSystem, since, policyResult)
}

func GuestPendingDeleteUsage(userToken mcclient.TokenCredential, prefix string, scope rbacscope.TRbacScope, cred mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string,
	brands []string, cloudEnv string, includeSystem bool, since *time.Time, policyResult rbacutils.SPolicyResult) Usage {
	return guestUsage(userToken, prefix, scope, cred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, nil, true, includeSystem, since, policyResult)
}

func GuestRunningUsage(userToken mcclient.TokenCredential, prefix string, scope rbacscope.TRbacScope, cred mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string,
	brands []string, cloudEnv string, includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) Usage {
	return guestUsage(userToken, prefix, scope, cred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, []string{api.VM_RUNNING}, false, includeSystem, nil, policyResult)
}

func GuestReadyUsage(userToken mcclient.TokenCredential, prefix string, scope rbacscope.TRbacScope, cred mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string,
	brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) Usage {
	return guestUsage(userToken, prefix, scope, cred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, []string{api.VM_READY}, false, includeSystem, nil, policyResult)
}

func guestHypervisorsUsage(
	userToken mcclient.TokenCredential,
	prefix string,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string,
	status, hypervisors []string,
	pendingDelete, includeSystem bool,
	since *time.Time,
	policyResult rbacutils.SPolicyResult,
) Usage {
	count := make(map[string]interface{})

	results := db.UsagePolicyCheck(userToken, models.GuestManager, scope)
	log.Debugf("guestHypervisorsUsage origin %s", results.String())
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	log.Debugf("guestHypervisorsUsage policyResults %s results %s", policyResult.String(), results.String())
	// temporarily hide system resources
	// XXX needs more work later
	guest := models.GuestManager.TotalCount(scope, ownerId, rangeObjs, status, hypervisors,
		includeSystem, pendingDelete, hostTypes, resourceTypes, providers, brands, cloudEnv, since,
		results,
	)

	count[prefix] = guest.TotalGuestCount
	count[fmt.Sprintf("%s.any_pool", prefix)] = guest.TotalGuestCount
	count[fmt.Sprintf("%s.cpu", prefix)] = guest.TotalCpuCount
	count[fmt.Sprintf("%s.memory", prefix)] = guest.TotalMemSize

	if len(hypervisors) == 1 && hypervisors[0] == api.HYPERVISOR_POD {
		return count
	}

	count[fmt.Sprintf("%s.disk", prefix)] = guest.TotalDiskSize
	count[fmt.Sprintf("%s.isolated_devices", prefix)] = guest.TotalIsolatedCount

	count[fmt.Sprintf("%s.ha", prefix)] = guest.TotalBackupGuestCount
	count[fmt.Sprintf("%s.ha.cpu", prefix)] = guest.TotalBackupCpuCount
	count[fmt.Sprintf("%s.ha.memory", prefix)] = guest.TotalBackupMemSize
	count[fmt.Sprintf("%s.ha.disk", prefix)] = guest.TotalBackupDiskSize

	return count
}

func guestUsage(userToken mcclient.TokenCredential, prefix string, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string,
	status []string, pendingDelete, includeSystem bool, since *time.Time,
	policyResult rbacutils.SPolicyResult,
) Usage {
	hypervisors := sets.NewString(api.HYPERVISORS...)
	hypervisors.Delete(api.HYPERVISOR_POD, api.HYPERVISOR_BAREMETAL)
	return guestHypervisorsUsage(userToken, prefix, scope, userCred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, status, hypervisors.List(), pendingDelete, includeSystem, since, policyResult)
}

/*func containerUsage(prefix string, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	hypervisors := []string{api.HYPERVISOR_CONTAINER}
	return guestHypervisorsUsage(prefix, scope, userCred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, nil, hypervisors, false)
}*/

func IsolatedDeviceUsage(userToken mcclient.TokenCredential, pref string, scope rbacscope.TRbacScope, userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostType []string, resourceTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	prefix := "isolated_devices"
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.IsolatedDeviceManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		return count
	}
	ret, _ := models.IsolatedDeviceManager.TotalCount(scope, userCred, hostType, resourceTypes, providers, brands, cloudEnv, rangeObjs, results)
	count[prefix] = ret.Devices
	return count
}

func getSysKey(scope rbacscope.TRbacScope, key string) string {
	return _getKey(scope, key, true)
}

func getKey(scope rbacscope.TRbacScope, key string) string {
	return _getKey(scope, key, false)
}

func _getKey(scope rbacscope.TRbacScope, key string, includeSystem bool) string {
	switch scope {
	case rbacscope.ScopeProject:
		if includeSystem {
			if len(key) > 0 {
				return fmt.Sprintf("project.%s", key)
			} else {
				return "project"
			}
		} else {
			return key
		}
	case rbacscope.ScopeDomain:
		if len(key) > 0 {
			return fmt.Sprintf("domain.%s", key)
		} else {
			return "domain"
		}
	default:
		if includeSystem {
			return key
		} else {
			if len(key) > 0 {
				return fmt.Sprintf("all.%s", key)
			} else {
				return "all"
			}
		}
	}
}

func EipUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.ElasticipManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	eipUsage := models.ElasticipManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "eip")] = eipUsage.Total()
	count[getKey(scope, "eip.public_ip")] = eipUsage.PublicIPCount
	count[getKey(scope, "eip.public_ip.bandwidth.mb")] = eipUsage.PublicIpBandwidth
	count[getKey(scope, "eip.floating_ip")] = eipUsage.EIPCount
	count[getKey(scope, "eip.floating_ip.bandwidth.mb")] = eipUsage.EipBandwidth
	count[getKey(scope, "eip.floating_ip")] = eipUsage.EIPCount
	count[getKey(scope, "eip.floating_ip.used")] = eipUsage.EIPUsedCount
	count[getKey(scope, "eip.used")] = eipUsage.EIPUsedCount + eipUsage.PublicIPCount
	return count
}

func BucketUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})

	results := db.UsagePolicyCheck(userToken, models.BucketManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}

	bucketUsage := models.BucketManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "buckets")] = bucketUsage.Buckets
	count[getKey(scope, "bucket_objects")] = bucketUsage.Objects
	count[getKey(scope, "bucket_bytes")] = bucketUsage.Bytes
	count[getKey(scope, "bucket_bytes_limit")] = bucketUsage.BytesLimit
	count[getKey(scope, "bucket_disk_used_rate")] = bucketUsage.DiskUsedRate
	return count
}

func SnapshotUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.SnapshotManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	cnt, _ := models.TotalSnapshotCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "snapshot")] = cnt
	return count
}

func InstanceSnapshotUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.InstanceSnapshotManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	cnt, _ := models.TotalInstanceSnapshotCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "instance_snapshot")] = cnt
	return count
}

func LoadbalancerUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.LoadbalancerManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	cnt, _ := models.LoadbalancerManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "loadbalancer")] = cnt
	return count
}

func DBInstanceUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.DBInstanceManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	cnt, _ := models.DBInstanceManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "rds")] = cnt.TotalRdsCount
	count[getKey(scope, "rds.cpu")] = cnt.TotalCpuCount
	count[getKey(scope, "rds.memory")] = cnt.TotalMemSizeMb
	count[getKey(scope, "rds.disk_size_gb")] = cnt.TotalDiskSizeGb
	count[getKey(scope, "rds.disk_size_used_mb")] = cnt.TotalDiskSizeUsedMb
	count[getKey(scope, "rds.disk_size_used_rate")] = cnt.DiskUsedRate
	return count
}

func MongoDBUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.MongoDBManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	cnt, _ := models.MongoDBManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "mongodb")] = cnt.TotalMongodbCount
	count[getKey(scope, "mongodb.cpu")] = cnt.TotalCpuCount
	count[getKey(scope, "mongodb.memory")] = cnt.TotalMemSizeMb
	return count
}

func ElasticSearchUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.ElasticSearchManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	cnt, _ := models.ElasticSearchManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "es")] = cnt.TotalEsCount
	count[getKey(scope, "es.cpu")] = cnt.TotalCpuCount
	count[getKey(scope, "es.memory")] = cnt.TotalMemSizeGb * 1024
	return count
}

func KafkaUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.KafkaManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	cnt, _ := models.KafkaManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "kafka")] = cnt.TotalKafkaCount
	count[getKey(scope, "kafka.disk")] = cnt.TotalDiskSizeGb
	return count
}

func ElasticCacheUsage(userToken mcclient.TokenCredential, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	results := db.UsagePolicyCheck(userToken, models.ElasticcacheManager, scope)
	results = results.Merge(policyResult)
	if results.Result.IsDeny() {
		// deny
		return count
	}
	cnt, _ := models.ElasticcacheManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, results)
	count[getKey(scope, "cache")] = cnt
	return count
}
