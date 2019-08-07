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

type objUsageFunc func(rbacutils.TRbacScope, mcclient.IIdentityProvider, db.IStandaloneModel, []string, []string, []string, string) (Usage, error)

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
			httperrors.NotFoundError(w, err.Error())
			return
		}
		ownerId, scope, err := db.FetchUsageOwnerScope(ctx, userCred, getQuery(r))
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
		log.Debugf("%s %s", ownerId, scope)
		query := getQuery(r)
		hostTypes := json.GetQueryStringArray(query, "host_type")
		// resourceTypes := json.GetQueryStringArray(query, "resource_type")
		providers := json.GetQueryStringArray(query, "provider")
		brands := json.GetQueryStringArray(query, "brand")
		cloudEnv, _ := query.GetString("cloud_env")
		usage, err := reporter(scope, ownerId, obj, hostTypes, providers, brands, cloudEnv)
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
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

func ReportHostUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, host db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, host, hostTypes, providers, brands, cloudEnv)
}

func ReportWireUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, wire db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, wire, hostTypes, providers, brands, cloudEnv)
}

func ReportCloudAccountUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, account db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, account, hostTypes, providers, brands, cloudEnv)
}

func ReportCloudProviderUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, provider db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, provider, hostTypes, providers, brands, cloudEnv)
}

func ReportSchedtagUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, schedtag db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, schedtag, hostTypes, providers, brands, cloudEnv)
}

func ReportZoneUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, zone db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, zone, hostTypes, providers, brands, cloudEnv)
}

func ReportCloudRegionUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, cloudRegion db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (Usage, error) {
	return ReportGeneralUsage(scope, userCred, cloudRegion, hostTypes, providers, brands, cloudEnv)
}

func getAdminGeneralUsage(userCred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (count Usage, err error) {
	count = RegionUsage(providers, brands, cloudEnv)
	zone := ZoneUsage(providers, brands, cloudEnv)
	count.Include(zone)
	vpc := VpcUsage(providers, brands, cloudEnv)
	count.Include(vpc)

	var pmemTotal float64
	var pcpuTotal float64

	hostEnabledUsage := HostEnabledUsage("", userCred, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv)
	pmemTotal = float64(hostEnabledUsage.Get("enabled_hosts.memory").(int64))
	pcpuTotal = float64(hostEnabledUsage.Get("enabled_hosts.cpu").(int64))
	if rangeObj != nil && rangeObj.Keyword() == "host" {
		host := rangeObj.(*models.SHost)
		pmemTotal = float64(host.MemSize)
		pcpuTotal = float64(host.CpuCount)
		count.Add("memory", host.MemSize)
		count.Add("cpu", host.CpuCount)
		count.Add("memory.virtual", host.GetVirtualMemorySize())
		count.Add("cpu.virtual", host.GetVirtualCPUCount())
	}

	guestRunningUsage := GuestRunningUsage("all.running_servers", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv)
	runningMem := guestRunningUsage.Get("all.running_servers.memory").(int)
	runningCpu := guestRunningUsage.Get("all.running_servers.cpu").(int)

	containerRunningUsage := containerUsage("all.containers", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, nil, providers, brands, cloudEnv)
	containerRunningMem := containerRunningUsage.Get("all.containers.memory").(int)
	containerRunningCpu := containerRunningUsage.Get("all.containers.cpu").(int)
	runningMem += containerRunningMem
	runningCpu += containerRunningCpu
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

	storageUsage := StorageUsage("", rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv)

	count.Include(
		HostAllUsage("", userCred, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		HostAllUsage("prepaid_pool", userCred, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		HostAllUsage("any_pool", userCred, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		hostEnabledUsage,
		HostEnabledUsage("prepaid_pool", userCred, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		HostEnabledUsage("any_pool", userCred, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		BaremetalUsage(userCred, rangeObj, hostTypes, providers, brands, cloudEnv),

		storageUsage,
		StorageUsage("prepaid_pool", rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		StorageUsage("any_pool", rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		GuestNormalUsage("all.servers", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		GuestNormalUsage("all.servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		GuestNormalUsage("all.servers.any_pool", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		GuestPendingDeleteUsage("all.pending_delete_servers", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		GuestPendingDeleteUsage("all.pending_delete_servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		GuestPendingDeleteUsage("all.pending_delete_servers.any_pool", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		GuestReadyUsage("all.ready_servers", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		GuestReadyUsage("all.ready_servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		GuestReadyUsage("all.ready_servers.any_pool", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		guestRunningUsage,
		GuestRunningUsage("all.running_servers.prepaid_pool", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		GuestRunningUsage("all.running_servers.any_pool", rbacutils.ScopeSystem, nil, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		containerRunningUsage,

		IsolatedDeviceUsage("", rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		IsolatedDeviceUsage("prepaid_pool", rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		IsolatedDeviceUsage("any_pool", rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		WireUsage(rangeObj, hostTypes, providers, brands, cloudEnv),

		NetworkUsage("all", rbacutils.ScopeSystem, nil, providers, brands, cloudEnv, rangeObj),

		EipUsage(rbacutils.ScopeSystem, nil, rangeObj, providers, brands, cloudEnv),

		BucketUsage(rbacutils.ScopeSystem, nil, rangeObj, providers, brands, cloudEnv),

		SnapshotUsage(rbacutils.ScopeSystem, nil, rangeObj, providers, brands, cloudEnv),
	)

	return
}

func getCommonGeneralUsage(scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (count Usage, err error) {
	guestNormalUsage := GuestNormalUsage("servers", scope, cred, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv)

	containerUsage := containerUsage("containers", scope, cred, rangeObj, hostTypes, nil, providers, brands, cloudEnv)

	eipUsage := EipUsage(scope, cred, rangeObj, providers, brands, cloudEnv)

	bucketUsage := BucketUsage(scope, cred, rangeObj, providers, brands, cloudEnv)

	snapshotUsage := SnapshotUsage(scope, cred, rangeObj, providers, brands, cloudEnv)

	disksUsage := disksUsage("", rangeObj, nil, nil, providers, brands, cloudEnv, scope, cred)

	nicsUsage := nicsUsage(rangeObj, nil, providers, brands, cloudEnv, scope, cred)

	count = guestNormalUsage.Include(
		GuestNormalUsage("servers.prepaid_pool", scope, cred, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		GuestNormalUsage("servers.any_pool", scope, cred, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		GuestRunningUsage("running_servers", scope, cred, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		GuestRunningUsage("running_servers.prepaid_pool", scope, cred, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		GuestRunningUsage("running_servers.any_pool", scope, cred, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		GuestPendingDeleteUsage("pending_delete_servers", scope, cred, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		GuestPendingDeleteUsage("pending_delete_servers.prepaid_pool", scope, cred, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		GuestPendingDeleteUsage("pending_delete_servers.any_pool", scope, cred, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		GuestReadyUsage("ready_servers", scope, cred, rangeObj, hostTypes, []string{api.HostResourceTypeShared}, providers, brands, cloudEnv),
		GuestReadyUsage("ready_servers.prepaid_pool", scope, cred, rangeObj, hostTypes, []string{api.HostResourceTypePrepaidRecycle}, providers, brands, cloudEnv),
		GuestReadyUsage("ready_servers.any_pool", scope, cred, rangeObj, hostTypes, nil, providers, brands, cloudEnv),

		containerUsage,

		NetworkUsage("", scope, cred, providers, brands, cloudEnv, rangeObj),

		eipUsage,

		bucketUsage,

		snapshotUsage,

		disksUsage,

		nicsUsage,
	)
	return
}

func ReportGeneralUsage(scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) (count Usage, err error) {
	count = make(map[string]interface{})

	if scope == rbacutils.ScopeSystem {
		count, err = getAdminGeneralUsage(userCred, rangeObj, hostTypes, providers, brands, cloudEnv)
		if err != nil {
			return
		}
	}

	if scope.HigherEqual(rbacutils.ScopeDomain) {
		commonUsage, err := getCommonGeneralUsage(rbacutils.ScopeDomain, userCred, rangeObj, hostTypes, providers, brands, cloudEnv)
		if err == nil {
			count.Include(commonUsage)
		}
	} else if scope.HigherEqual(rbacutils.ScopeProject) {
		commonUsage, err := getCommonGeneralUsage(rbacutils.ScopeProject, userCred, rangeObj, hostTypes, providers, brands, cloudEnv)
		if err == nil {
			count.Include(commonUsage)
		}
	}
	return
}

func RegionUsage(providers []string, brands []string, cloudEnv string) Usage {
	q := models.CloudregionManager.Query()

	if len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		subq := models.VpcManager.Query("cloudregion_id")
		subq = models.CloudProviderFilter(subq, subq.Field("manager_id"), providers, brands, cloudEnv)
		q = q.In("id", subq.SubQuery())
	}

	count := make(map[string]interface{})
	count["regions"], _ = q.CountWithError()
	return count
}

func ZoneUsage(providers []string, brands []string, cloudEnv string) Usage {
	q := models.ZoneManager.Query()

	if len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		subq := models.HostManager.Query("zone_id")
		subq = models.CloudProviderFilter(subq, subq.Field("manager_id"), providers, brands, cloudEnv)
		q = q.In("id", subq.SubQuery())
	}

	count := make(map[string]interface{})
	count["zones"], _ = q.CountWithError()
	return count
}

func VpcUsage(providers []string, brands []string, cloudEnv string) Usage {
	q := models.VpcManager.Query()
	if len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		q = models.CloudProviderFilter(q, q.Field("manager_id"), providers, brands, cloudEnv)
	}

	count := make(map[string]interface{})
	count["vpcs"], _ = q.CountWithError()
	return count
}

func StorageUsage(prefix string, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	sPrefix := "storages"
	dPrefix := "all.disks"
	if len(prefix) > 0 {
		sPrefix = fmt.Sprintf("%s.%s", sPrefix, prefix)
		dPrefix = fmt.Sprintf("%s.%s", dPrefix, prefix)
	}
	count := make(map[string]interface{})
	result := models.StorageManager.TotalCapacity(rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, rbacutils.ScopeSystem, nil)
	count[sPrefix] = result.Capacity
	count[fmt.Sprintf("%s.virtual", sPrefix)] = result.CapacityVirtual
	count[dPrefix] = result.CapacityUsed
	count[fmt.Sprintf("%s.unready", dPrefix)] = result.CapacityUnready
	count[fmt.Sprintf("%s.attached", dPrefix)] = result.AttachedCapacity
	count[fmt.Sprintf("%s.detached", dPrefix)] = result.DetachedCapacity

	storageCmtRate := 0.0
	if result.Capacity > 0 {
		storageCmtRate = utils.FloatRound(float64(result.CapacityUsed)/float64(result.Capacity), 2)
	}
	count[fmt.Sprintf("%s.commit_rate", sPrefix)] = storageCmtRate

	return count
}

func disksUsage(prefix string, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) Usage {
	dPrefix := "disks"
	if len(prefix) > 0 {
		dPrefix = fmt.Sprintf("%s.%s", dPrefix, prefix)
	}
	count := make(map[string]interface{})
	result := models.StorageManager.TotalCapacity(rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, scope, ownerId)
	count[dPrefix] = result.CapacityUsed
	count[fmt.Sprintf("%s.unready", dPrefix)] = result.CapacityUnready
	count[fmt.Sprintf("%s.attached", dPrefix)] = result.AttachedCapacity
	count[fmt.Sprintf("%s.detached", dPrefix)] = result.DetachedCapacity

	return count
}

func WireUsage(rangeObj db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	count := make(map[string]interface{})
	result := models.WireManager.TotalCount(rangeObj, hostTypes, providers, brands, cloudEnv, rbacutils.ScopeSystem, nil)
	count["wires"] = result.WiresCount
	count["networks"] = result.NetCount
	count["all.nics.guest"] = result.GuestNicCount
	count["all.nics.host"] = result.HostNicCount
	count["all.nics.reserve"] = result.ReservedCount
	count["all.nics.group"] = result.GroupNicCount
	count["all.nics.lb"] = result.LbNicCount
	count["all.nics"] = result.NicCount()
	return count
}

func nicsUsage(rangeObj db.IStandaloneModel, hostTypes []string, providers []string, brands []string, cloudEnv string, scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider) Usage {
	count := make(map[string]interface{})
	result := models.WireManager.TotalCount(rangeObj, hostTypes, providers, brands, cloudEnv, scope, ownerId)
	count["nics.guest"] = result.GuestNicCount
	count["nics.group"] = result.GroupNicCount
	count["nics.lb"] = result.LbNicCount
	count["nics"] = result.GuestNicCount + result.GroupNicCount + result.LbNicCount
	return count
}

func prefixKey(prefix string, key string) string {
	if len(prefix) > 0 {
		return prefix + "." + key
	} else {
		return key
	}
}

func NetworkUsage(prefix string, scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, providers []string, brands []string, cloudEnv string, rangeObj db.IStandaloneModel) Usage {
	count := make(map[string]interface{})
	ret := models.NetworkManager.TotalPortCount(scope, userCred, providers, brands, cloudEnv, rangeObj)
	count[prefixKey(prefix, "ports")] = ret.Count
	count[prefixKey(prefix, "ports_exit")] = ret.CountExt
	return count
}

func HostAllUsage(pref string, userCred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	prefix := "hosts"
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	return hostUsage(userCred, prefix, rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, tristate.None, tristate.None)
}

func HostEnabledUsage(pref string, userCred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	prefix := "enabled_hosts"
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	return hostUsage(userCred, prefix, rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, tristate.True, tristate.None)
}

func BaremetalUsage(userCred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel,
	hostTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	prefix := "baremetals"
	count := hostUsage(userCred, prefix, rangeObj, hostTypes, nil, providers, brands, cloudEnv, tristate.None, tristate.True)
	delete(count, fmt.Sprintf("%s.memory.virtual", prefix))
	delete(count, fmt.Sprintf("%s.cpu.virtual", prefix))
	return count
}

func hostUsage(
	userCred mcclient.IIdentityProvider, prefix string,
	rangeObj db.IStandaloneModel, hostTypes []string,
	resourceTypes []string, providers []string, brands []string, cloudEnv string,
	enabled, isBaremetal tristate.TriState,
) Usage {
	count := make(map[string]interface{})

	result := models.HostManager.TotalCount(userCred, rangeObj, "", "", hostTypes, resourceTypes, providers, brands, cloudEnv, enabled, isBaremetal)
	count[prefix] = result.Count
	count[fmt.Sprintf("%s.memory", prefix)] = result.Memory
	count[fmt.Sprintf("%s.cpu", prefix)] = result.CPU
	count[fmt.Sprintf("%s.memory.virtual", prefix)] = result.MemoryVirtual
	count[fmt.Sprintf("%s.cpu.virtual", prefix)] = result.CPUVirtual

	return count
}

func GuestNormalUsage(prefix string, scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	return guestUsage(prefix, scope, cred, rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, nil, false)
}

func GuestPendingDeleteUsage(prefix string, scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	return guestUsage(prefix, scope, cred, rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, nil, true)
}

func GuestRunningUsage(prefix string, scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	return guestUsage(prefix, scope, cred, rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, []string{api.VM_RUNNING}, false)
}

func GuestReadyUsage(prefix string, scope rbacutils.TRbacScope, cred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	return guestUsage(prefix, scope, cred, rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, []string{api.VM_READY}, false)
}

func guestHypervisorsUsage(
	prefix string,
	scope rbacutils.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string,
	status, hypervisors []string,
	pendingDelete bool,
) Usage {
	// temporarily hide system resources
	// XXX needs more work later
	guest := models.GuestManager.TotalCount(scope, ownerId, rangeObj, status, hypervisors, false, pendingDelete, hostTypes, resourceTypes, providers, brands, cloudEnv)
	count := make(map[string]interface{})
	count[prefix] = guest.TotalGuestCount
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

func guestUsage(prefix string, scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string,
	status []string, pendingDelete bool) Usage {
	hypervisors := sets.NewString(api.HYPERVISORS...)
	hypervisors.Delete(api.HYPERVISOR_CONTAINER)
	return guestHypervisorsUsage(prefix, scope, userCred, rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, status, hypervisors.List(), pendingDelete)
}

func containerUsage(prefix string, scope rbacutils.TRbacScope, userCred mcclient.IIdentityProvider, rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	hypervisors := []string{api.HYPERVISOR_CONTAINER}
	return guestHypervisorsUsage(prefix, scope, userCred, rangeObj, hostTypes, resourceTypes, providers, brands, cloudEnv, nil, hypervisors, false)
}

func IsolatedDeviceUsage(pref string, rangeObj db.IStandaloneModel, hostType []string, resourceTypes []string, providers []string, brands []string, cloudEnv string) Usage {
	prefix := "isolated_devices"
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	ret, _ := models.IsolatedDeviceManager.TotalCount(hostType, resourceTypes, providers, brands, cloudEnv, rangeObj)
	count := make(map[string]interface{})
	count[prefix] = ret.Devices
	return count
}

func getKey(projectId, key string) string {
	if len(projectId) > 0 {
		return key
	} else {
		return fmt.Sprintf("all.%s", key)
	}
}

func EipUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, providers []string, brands []string, cloudEnv string) Usage {
	projectId := mcclient.OwnerIdString(ownerId, scope)
	eipUsage := models.ElasticipManager.TotalCount(scope, ownerId, rangeObj, providers, brands, cloudEnv)
	count := make(map[string]interface{})
	count[getKey(projectId, "eip")] = eipUsage.Total()
	count[getKey(projectId, "eip.public_ip")] = eipUsage.PublicIPCount
	count[getKey(projectId, "eip.floating_ip")] = eipUsage.EIPCount
	count[getKey(projectId, "eip.floating_ip.used")] = eipUsage.EIPUsedCount
	count[getKey(projectId, "eip.used")] = eipUsage.EIPUsedCount + eipUsage.PublicIPCount
	return count
}

func BucketUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, providers []string, brands []string, cloudEnv string) Usage {
	projectId := mcclient.OwnerIdString(ownerId, scope)
	bucketUsage := models.BucketManager.TotalCount(scope, ownerId, rangeObj, providers, brands, cloudEnv)
	count := make(map[string]interface{})
	count[getKey(projectId, "buckets")] = bucketUsage.Buckets
	count[getKey(projectId, "bucket_objects")] = bucketUsage.Objects
	count[getKey(projectId, "bucket_bytes")] = bucketUsage.Bytes
	return count
}

func SnapshotUsage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, rangeObj db.IStandaloneModel, providers []string, brands []string, cloudEnv string) Usage {
	projectId := mcclient.OwnerIdString(ownerId, scope)
	cnt, _ := models.TotalSnapshotCount(scope, ownerId, rangeObj, providers, brands, cloudEnv)
	count := make(map[string]interface{})
	count[getKey(projectId, "snapshot")] = cnt
	return count
}
