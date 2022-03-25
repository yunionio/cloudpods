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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/tagutils"
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

type objUsageFunc func(rbacutils.TRbacScope, mcclient.IIdentityProvider, bool, []db.IStandaloneModel, []string, []string, []string, string, bool, rbacutils.SPolicyResult) (Usage, error)

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
		projectTags := &tagutils.TTagSetList{}
		getQuery(r).Unmarshal(projectTags, "project_tags")
		for i := range result.ProjectTags {
			projectTags.Append(result.ProjectTags[i])
		}
		result.ProjectTags = *projectTags
		isOwner := false
		if scope == rbacutils.ScopeDomain && obj != nil && db.IsObjectRbacAllowed(ctx, obj, userCred, policy.PolicyActionGet, "usage") == nil {
			isOwner = true
		}
		log.Debugf("ownerId: %s isOwner: %v scope: %s result: %s", ownerId, isOwner, scope, result.String())
		query := getQuery(r)
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
		key := getCacheKey(scope, ownerId, isOwner, rangeObjs, hostTypes, providers, brands, cloudEnv, includeSystem, *projectTags)
		if !refresh {
			cached := usageCache.Get(key)
			if cached != nil {
				response(w, cached)
				return
			}
		}
		usage, err := reporter(scope, ownerId, isOwner, rangeObjs, hostTypes, providers, brands, cloudEnv, includeSystem, result)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		usageCache.AtomicSet(key, usage)
		response(w, usage)
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

func response(w http.ResponseWriter, obj interface{}) {
	body := map[string]interface{}{
		"usage": obj,
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

func ReportHostUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, hosts []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, isOwner, hosts, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportWireUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, wires []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, isOwner, wires, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportCloudAccountUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, accounts []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, isOwner, accounts, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportCloudProviderUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, managers []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, isOwner, managers, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportSchedtagUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, schedtags []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, isOwner, schedtags, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportZoneUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, zones []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, isOwner, zones, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func ReportCloudRegionUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, isOwner bool, cloudRegions []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, isOwner, cloudRegions, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
}

func getSystemGeneralUsage(
	userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostTypes []string,
	providers []string, brands []string, cloudEnv string, includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) (Usage, error) {
	count := RegionUsage(rangeObjs, providers, brands, cloudEnv)
	zone := ZoneUsage(rangeObjs, providers, brands, cloudEnv)
	count.Include(zone)

	var pmemTotal float64
	var pcpuTotal float64

	hostEnabledUsage := HostEnabledUsage("", userCred, rbacutils.ScopeSystem, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult)
	pmemTotal = float64(hostEnabledUsage.Get("enabled_hosts.memory").(int64))
	pcpuTotal = float64(hostEnabledUsage.Get("enabled_hosts.cpu").(int64))
	if len(rangeObjs) > 0 && rangeObjs[0].Keyword() == "host" {
		host := rangeObjs[0].(*models.SHost)
		pmemTotal = float64(host.MemSize)
		pcpuTotal = float64(host.CpuCount)
		count.Add("memory", host.MemSize)
		count.Add("cpu", host.CpuCount)
		count.Add("memory.virtual", int64(host.GetVirtualMemorySize()))
		count.Add("cpu.virtual", int64(host.GetVirtualCPUCount()))
	}

	guestRunningUsage := GuestRunningUsage("all.running_servers", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, policyResult)
	runningMem := guestRunningUsage.Get("all.running_servers.memory").(int)
	runningCpu := guestRunningUsage.Get("all.running_servers.cpu").(int)

	// containerRunningUsage := containerUsage("all.containers", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv)
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
		VpcUsage("all", providers, brands, cloudEnv, nil, rbacutils.ScopeSystem, rangeObjs, policyResult),

		DnsZoneUsage("", nil, rbacutils.ScopeSystem, policyResult),

		HostAllUsage("", userCred, rbacutils.ScopeSystem, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult),
		// HostAllUsage("prepaid_pool", userCred, rbacutils.ScopeSystem, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// HostAllUsage("any_pool", userCred, rbacutils.ScopeSystem, rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		hostEnabledUsage,
		// HostEnabledUsage("prepaid_pool", userCred, rbacutils.ScopeSystem, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// HostEnabledUsage("any_pool", userCred, rbacutils.ScopeSystem, rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		BaremetalUsage(userCred, rbacutils.ScopeSystem, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),

		StorageUsage("", rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, includeSystem, rbacutils.ScopeSystem, nil, policyResult),
		StorageUsage("system", rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, true, rbacutils.ScopeSystem, nil, policyResult),
		// StorageUsage("prepaid_pool", rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false, includeSystem, rbacutils.ScopeSystem, nil),
		// StorageUsage("any_pool", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false, includeSystem, rbacutils.ScopeSystem, nil),
		// StorageUsage("any_pool.system", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false, true, rbacutils.ScopeSystem, nil),
		// StorageUsage("any_pool.pending_delete", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, true, includeSystem, rbacutils.ScopeSystem, nil),
		// StorageUsage("any_pool.pending_delete.system", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, true, true, rbacutils.ScopeSystem, nil),

		GuestNormalUsage("all.servers", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, nil, policyResult),
		GuestNormalUsage("all.servers.last_week", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, &lastWeek, policyResult),
		GuestNormalUsage("all.servers.last_month", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, &lastMonth, policyResult),
		// GuestNormalUsage("all.servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestNormalUsage("all.servers.any_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),

		GuestPendingDeleteUsage("all.pending_delete_servers", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, nil, policyResult),
		GuestPendingDeleteUsage("all.pending_delete_servers.last_week", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, &lastWeek, policyResult),
		GuestPendingDeleteUsage("all.pending_delete_servers.last_month", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, &lastMonth, policyResult),
		// GuestPendingDeleteUsage("all.pending_delete_servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestNormalUsage("all.servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestNormalUsage("all.servers.any_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),

		// GuestPendingDeleteUsage("all.pending_delete_servers", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem),
		// GuestPendingDeleteUsage("all.pending_delete_servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestPendingDeleteUsage("all.pending_delete_servers.any_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),

		GuestReadyUsage("all.ready_servers", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, includeSystem, policyResult),
		// GuestReadyUsage("all.ready_servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestReadyUsage("all.ready_servers.any_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),
		// GuestRunningUsage("all.running_servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, includeSystem),
		// GuestRunningUsage("all.running_servers.any_pool", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, includeSystem),

		guestRunningUsage,
		// containerRunningUsage,

		IsolatedDeviceUsage("", rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult),
		// IsolatedDeviceUsage("prepaid_pool", rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// IsolatedDeviceUsage("any_pool", rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		WireUsage(rbacutils.ScopeSystem, nil, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),
		NetworkUsage("all", rbacutils.ScopeSystem, nil, providers, brands, cloudEnv, rangeObjs, policyResult),

		EipUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		BucketUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		SnapshotUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		InstanceSnapshotUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		LoadbalancerUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		DBInstanceUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		MongoDBUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticSearchUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		KafkaUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticCacheUsage(rbacutils.ScopeSystem, nil, rangeObjs, providers, brands, cloudEnv, policyResult),
	)

	return count, nil
}

func getDomainGeneralUsage(scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) (Usage, error) {
	lastWeek := time.Now().Add(-7 * 24 * time.Hour)
	lastMonth := time.Now().Add(-30 * 24 * time.Hour)
	count := GuestNormalUsage(getKey(scope, "servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, nil, policyResult)

	var pmemTotal float64
	var pcpuTotal float64

	hostEnabledUsage := HostEnabledUsage("", cred, rbacutils.ScopeDomain, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult)
	pmemTotal = float64(hostEnabledUsage.Get("domain.enabled_hosts.memory").(int64))
	pcpuTotal = float64(hostEnabledUsage.Get("domain.enabled_hosts.cpu").(int64))

	guestRunningUsage := GuestRunningUsage("domain.running_servers", rbacutils.ScopeDomain, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, policyResult)
	runningMem := guestRunningUsage.Get("domain.running_servers.memory").(int)
	runningCpu := guestRunningUsage.Get("domain.running_servers.cpu").(int)

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
		VpcUsage("domain", providers, brands, cloudEnv, cred, rbacutils.ScopeDomain, rangeObjs, policyResult),

		DnsZoneUsage("domain", cred, rbacutils.ScopeDomain, policyResult),

		HostAllUsage("", cred, rbacutils.ScopeDomain, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult),
		// HostAllUsage("prepaid_pool", cred, rbacutils.ScopeDomain, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// HostAllUsage("any_pool", cred, rbacutils.ScopeDomain, rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		hostEnabledUsage,
		// HostEnabledUsage("prepaid_pool", cred, rbacutils.ScopeDomain, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		// HostEnabledUsage("any_pool", cred, rbacutils.ScopeDomain, rangeObjs, hostTypes, nil, providers, brands, cloudEnv),

		BaremetalUsage(cred, rbacutils.ScopeDomain, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),

		StorageUsage("", rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, false, rbacutils.ScopeDomain, cred, policyResult),
		StorageUsage("system", rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, true, rbacutils.ScopeDomain, cred, policyResult),
		// StorageUsage("prepaid_pool", rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false, false, rbacutils.ScopeDomain, cred),
		// StorageUsage("any_pool", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false, false, rbacutils.ScopeDomain, cred),
		// StorageUsage("any_pool.system", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false, true, rbacutils.ScopeDomain, cred),
		// StorageUsage("any_pool.pending_delete", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, true, false, rbacutils.ScopeDomain, cred),
		// StorageUsage("any_pool.pending_delete.system", rangeObjs, hostTypes, nil, providers, brands, cloudEnv, true, true, rbacutils.ScopeDomain, cred),

		GuestNormalUsage(getKey(scope, "servers.last_week"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastWeek, policyResult),
		GuestNormalUsage(getKey(scope, "servers.last_month"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastMonth, policyResult),
		// GuestNormalUsage(getKey(scope, "servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestNormalUsage(getKey(scope, "servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		guestRunningUsage,
		// GuestRunningUsage(getKey(scope, "running_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		// GuestRunningUsage(getKey(scope, "running_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestRunningUsage(getKey(scope, "running_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, nil, policyResult),
		GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.last_week"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastWeek, policyResult),
		GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.last_month"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastMonth, policyResult),
		// GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		GuestReadyUsage(getKey(scope, "ready_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, policyResult),
		// GuestReadyUsage(getKey(scope, "ready_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestReadyUsage(getKey(scope, "ready_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		WireUsage(scope, cred, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),
		NetworkUsage(getKey(scope, ""), scope, cred, providers, brands, cloudEnv, rangeObjs, policyResult),

		IsolatedDeviceUsage("", scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, policyResult),

		EipUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		BucketUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		// nicsUsage("domain", rangeObjs, hostTypes, providers, brands, cloudEnv, scope, cred),

		SnapshotUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		InstanceSnapshotUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		LoadbalancerUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		DBInstanceUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		MongoDBUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticSearchUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		KafkaUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticCacheUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),
	)
	return count, nil
}

func getProjectGeneralUsage(scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) (Usage, error) {
	lastWeek := time.Now().Add(-7 * 24 * time.Hour)
	lastMonth := time.Now().Add(-30 * 24 * time.Hour)
	count := GuestNormalUsage(getKey(scope, "servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, nil, policyResult)

	count.Include(
		GuestNormalUsage(getKey(scope, "servers.last_week"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastWeek, policyResult),
		GuestNormalUsage(getKey(scope, "servers.last_month"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastMonth, policyResult),
		// GuestNormalUsage(getKey(scope, "servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestNormalUsage(getKey(scope, "servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),
		GuestRunningUsage(getKey(scope, "running_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, policyResult),
		// GuestRunningUsage(getKey(scope, "running_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestRunningUsage(getKey(scope, "running_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, nil, policyResult),
		GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.last_week"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastWeek, policyResult),
		GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.last_month"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, &lastMonth, policyResult),
		// GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestPendingDeleteUsage(getKey(scope, "pending_delete_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		GuestReadyUsage(getKey(scope, "ready_servers"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv, false, policyResult),
		// GuestReadyUsage(getKey(scope, "ready_servers.prepaid_pool"), scope, cred, rangeObjs, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv, false),
		// GuestReadyUsage(getKey(scope, "ready_servers.any_pool"), scope, cred, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, false),

		WireUsage(scope, cred, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult),
		NetworkUsage(getKey(scope, ""), scope, cred, providers, brands, cloudEnv, rangeObjs, policyResult),

		EipUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		BucketUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		DisksUsage(getKey(scope, "disks"), rangeObjs, hostTypes, nil, providers, brands, cloudEnv, scope, cred, false, false, policyResult),
		DisksUsage(getKey(scope, "disks.system"), rangeObjs, hostTypes, nil, providers, brands, cloudEnv, scope, cred, false, true, policyResult),
		DisksUsage(getKey(scope, "pending_delete_disks"), rangeObjs, hostTypes, nil, providers, brands, cloudEnv, scope, cred, true, false, policyResult),
		DisksUsage(getKey(scope, "pending_delete_disks.system"), rangeObjs, hostTypes, nil, providers, brands, cloudEnv, scope, cred, true, true, policyResult),

		// nicsUsage("", rangeObjs, hostTypes, providers, brands, cloudEnv, scope, cred),

		SnapshotUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		InstanceSnapshotUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		LoadbalancerUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		DBInstanceUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		MongoDBUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticSearchUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		KafkaUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),

		ElasticCacheUsage(scope, cred, rangeObjs, providers, brands, cloudEnv, policyResult),
	)

	return count, nil
}

func ReportGeneralUsage(
	scope rbacutils.TRbacScope,
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

	// if scope == rbacutils.ScopeSystem || isOwner {
	if scope == rbacutils.ScopeSystem {
		count, err = getSystemGeneralUsage(userCred, rangeObjs, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
		if err != nil {
			return
		}
	}

	// if scope.HigherEqual(rbacutils.ScopeDomain) && len(userCred.GetProjectDomainId()) > 0 {
	if scope == rbacutils.ScopeDomain && len(userCred.GetProjectDomainId()) > 0 {
		commonUsage, err := getDomainGeneralUsage(rbacutils.ScopeDomain, userCred, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult)
		if err == nil {
			count.Include(commonUsage)
		}
	}

	// if scope.HigherEqual(rbacutils.ScopeProject) && len(userCred.GetProjectId()) > 0 {
	if scope == rbacutils.ScopeProject && len(userCred.GetProjectId()) > 0 {
		commonUsage, err := getProjectGeneralUsage(rbacutils.ScopeProject, userCred, rangeObjs, hostTypes, providers, brands, cloudEnv, policyResult)
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

func VpcUsage(prefix string, providers []string, brands []string, cloudEnv string, ownerId mcclient.IIdentityProvider, scope rbacutils.TRbacScope, rangeObjs []db.IStandaloneModel, policyResult rbacutils.SPolicyResult) Usage {
	q := models.VpcManager.Query().IsFalse("is_emulated")
	if len(rangeObjs) > 0 || len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		q = models.CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
		q = models.RangeObjectsFilter(q, rangeObjs, nil, nil, q.Field("manager_id"), nil, nil)
	}
	if scope == rbacutils.ScopeDomain {
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}

	q = db.ObjectIdQueryWithPolicyResult(q, models.VpcManager, policyResult)

	count := make(map[string]interface{})
	key := "vpcs"
	if len(prefix) > 0 {
		key = fmt.Sprintf("%s.vpcs", prefix)
	}
	count[key], _ = q.CountWithError()
	return count
}

func DnsZoneUsage(prefix string, ownerId mcclient.IIdentityProvider, scope rbacutils.TRbacScope, policyResult rbacutils.SPolicyResult) Usage {
	q := models.DnsZoneManager.Query()
	if scope == rbacutils.ScopeDomain {
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}

	q = db.ObjectIdQueryWithPolicyResult(q, models.DnsZoneManager, policyResult)

	count := make(map[string]interface{})
	key := "dns_zones"
	if len(prefix) > 0 {
		key = fmt.Sprintf("%s.dns_zones", prefix)
	}
	count[key], _ = q.CountWithError()
	return count
}

func StorageUsage(
	prefix string,
	rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	pendingDeleted bool, includeSystem bool,
	scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider,
	policyResult rbacutils.SPolicyResult,
) Usage {
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
		policyResult,
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
	for s, capa := range result.StorageTypeCapacity {
		count[fmt.Sprintf("%s.storage_type.%s", dPrefix, s)] = capa
	}
	for m, capa := range result.MediumeCapacity {
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
	dPrefix string,
	rangeObjs []db.IStandaloneModel,
	hostTypes []string,
	resourceTypes []string,
	providers []string,
	brands []string,
	cloudEnv string,
	scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider,
	pendingDeleted bool, includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) Usage {
	count := make(map[string]interface{})
	result := models.StorageManager.TotalCapacity(rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, scope, ownerId, pendingDeleted, includeSystem, false, policyResult)
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

func WireUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	result := models.WireManager.TotalCount(rangeObjs, hostTypes, providers, brands, cloudEnv, scope, userCred, policyResult)
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

/*func nicsUsage(prefix string, rangeObjs []db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) Usage {
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

func NetworkUsage(prefix string, scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, providers []string, brands []string, cloudEnv string, rangeObjs []db.IStandaloneModel, policyResult rbacutils.SPolicyResult) Usage {
	count := make(map[string]interface{})
	ret := models.NetworkManager.TotalPortCount(scope, userCred, providers, brands, cloudEnv, rangeObjs, policyResult)
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

func HostAllUsage(pref string, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope, rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	prefix := getSysKey(scope, "hosts")
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	return hostUsage(userCred, scope, prefix, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, tristate.None, tristate.False, policyResult)
}

func HostEnabledUsage(pref string, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope, rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	prefix := getSysKey(scope, "enabled_hosts")
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	return hostUsage(userCred, scope, prefix, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, tristate.True, tristate.False, policyResult)
}

func BaremetalUsage(userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope, rangeObjs []db.IStandaloneModel,
	hostTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	prefix := getSysKey(scope, "baremetals")
	count := hostUsage(userCred, scope, prefix, rangeObjs, hostTypes, nil, providers, brands, cloudEnv, tristate.None, tristate.True, policyResult)
	delete(count, fmt.Sprintf("%s.memory.virtual", prefix))
	delete(count, fmt.Sprintf("%s.cpu.virtual", prefix))
	return count
}

func hostUsage(
	userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope, prefix string,
	rangeObjs []db.IStandaloneModel, hostTypes []string,
	resourceTypes []string, providers []string, brands []string, cloudEnv string,
	enabled, isBaremetal tristate.TriState,
	policyResult rbacutils.SPolicyResult,
) Usage {
	count := make(map[string]interface{})

	result := models.HostManager.TotalCount(userCred, scope, rangeObjs, "", "", hostTypes, resourceTypes, providers, brands, cloudEnv, enabled, isBaremetal, policyResult)
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

	return count
}

func GuestNormalUsage(prefix string, scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string,
	brands []string, cloudEnv string, includeSystem bool, since *time.Time, policyResult rbacutils.SPolicyResult) Usage {
	return guestUsage(prefix, scope, cred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, nil, false, includeSystem, since, policyResult)
}

func GuestPendingDeleteUsage(prefix string, scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string,
	brands []string, cloudEnv string, includeSystem bool, since *time.Time, policyResult rbacutils.SPolicyResult) Usage {
	return guestUsage(prefix, scope, cred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, nil, true, includeSystem, since, policyResult)
}

func GuestRunningUsage(prefix string, scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string,
	brands []string, cloudEnv string, includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) Usage {
	return guestUsage(prefix, scope, cred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, []string{api.VM_RUNNING}, false, includeSystem, nil, policyResult)
}

func GuestReadyUsage(prefix string, scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string,
	brands []string, cloudEnv string, includeSystem bool, policyResult rbacutils.SPolicyResult) Usage {
	return guestUsage(prefix, scope, cred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, []string{api.VM_READY}, false, includeSystem, nil, policyResult)
}

func guestHypervisorsUsage(
	prefix string,
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string,
	status, hypervisors []string,
	pendingDelete, includeSystem bool,
	since *time.Time,
	policyResult rbacutils.SPolicyResult,
) Usage {
	// temporarily hide system resources
	// XXX needs more work later
	guest := models.GuestManager.TotalCount(scope, ownerId, rangeObjs, status, hypervisors,
		includeSystem, pendingDelete, hostTypes, resourceTypes, providers, brands, cloudEnv, since,
		policyResult,
	)
	count := make(map[string]interface{})
	count[prefix] = guest.TotalGuestCount
	count[fmt.Sprintf("%s.any_pool", prefix)] = guest.TotalGuestCount
	count[fmt.Sprintf("%s.cpu", prefix)] = guest.TotalCpuCount
	count[fmt.Sprintf("%s.memory", prefix)] = guest.TotalMemSize

	if len(hypervisors) == 1 && hypervisors[0] == api.HYPERVISOR_CONTAINER {
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

func guestUsage(prefix string, scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string,
	status []string, pendingDelete, includeSystem bool, since *time.Time,
	policyResult rbacutils.SPolicyResult,
) Usage {
	hypervisors := sets.NewString(api.HYPERVISORS...)
	hypervisors.Delete(api.HYPERVISOR_CONTAINER)
	return guestHypervisorsUsage(prefix, scope, userCred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, status, hypervisors.List(), pendingDelete, includeSystem, since, policyResult)
}

/*func containerUsage(prefix string, scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	hypervisors := []string{api.HYPERVISOR_CONTAINER}
	return guestHypervisorsUsage(prefix, scope, userCred, rangeObjs, hostTypes, resourceTypes, providers, brands, cloudEnv, nil, hypervisors, false)
}*/

func IsolatedDeviceUsage(pref string, scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, hostType []string, resourceTypes []string, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	prefix := "isolated_devices"
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	ret, _ := models.IsolatedDeviceManager.TotalCount(scope, userCred, hostType, resourceTypes, providers, brands, cloudEnv, rangeObjs, policyResult)
	count := make(map[string]interface{})
	count[prefix] = ret.Devices
	return count
}

func getSysKey(scope rbacutils.TRbacScope, key string) string {
	return _getKey(scope, key, true)
}

func getKey(scope rbacutils.TRbacScope, key string) string {
	return _getKey(scope, key, false)
}

func _getKey(scope rbacutils.TRbacScope, key string, includeSystem bool) string {
	switch scope {
	case rbacutils.ScopeProject:
		if includeSystem {
			if len(key) > 0 {
				return fmt.Sprintf("project.%s", key)
			} else {
				return "project"
			}
		} else {
			return key
		}
	case rbacutils.ScopeDomain:
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

func EipUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	eipUsage := models.ElasticipManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
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

func BucketUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	bucketUsage := models.BucketManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
	count[getKey(scope, "buckets")] = bucketUsage.Buckets
	count[getKey(scope, "bucket_objects")] = bucketUsage.Objects
	count[getKey(scope, "bucket_bytes")] = bucketUsage.Bytes
	return count
}

func SnapshotUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	cnt, _ := models.TotalSnapshotCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
	count[getKey(scope, "snapshot")] = cnt
	return count
}

func InstanceSnapshotUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	cnt, _ := models.TotalInstanceSnapshotCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
	count[getKey(scope, "instance_snapshot")] = cnt
	return count
}

func LoadbalancerUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	cnt, _ := models.LoadbalancerManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
	count[getKey(scope, "loadbalancer")] = cnt
	return count
}

func DBInstanceUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	cnt, _ := models.DBInstanceManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
	count[getKey(scope, "rds")] = cnt.TotalRdsCount
	count[getKey(scope, "rds.cpu")] = cnt.TotalCpuCount
	count[getKey(scope, "rds.memory")] = cnt.TotalMemSizeMb
	return count
}

func MongoDBUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	cnt, _ := models.MongoDBManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
	count[getKey(scope, "mongodb")] = cnt.TotalMongodbCount
	count[getKey(scope, "mongodb.cpu")] = cnt.TotalCpuCount
	count[getKey(scope, "mongodb.memory")] = cnt.TotalMemSizeMb
	return count
}

func ElasticSearchUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	cnt, _ := models.ElasticSearchManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
	count[getKey(scope, "es")] = cnt.TotalEsCount
	count[getKey(scope, "es.cpu")] = cnt.TotalCpuCount
	count[getKey(scope, "es.memory")] = cnt.TotalMemSizeGb * 1024
	return count
}

func KafkaUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	cnt, _ := models.KafkaManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
	count[getKey(scope, "kafka")] = cnt.TotalKafkaCount
	count[getKey(scope, "kafka.disk")] = cnt.TotalDiskSizeGb
	return count
}

func ElasticCacheUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObjs []db.IStandaloneModel, providers []string, brands []string, cloudEnv string, policyResult rbacutils.SPolicyResult) Usage {
	cnt, _ := models.ElasticcacheManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, cloudEnv, policyResult)
	count := make(map[string]interface{})
	count[getKey(scope, "cache")] = cnt
	return count
}
