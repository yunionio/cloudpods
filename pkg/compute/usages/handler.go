package usages

import (
	"context"
	"fmt"
	"net/http"

	json "yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
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

type objUsageFunc func(mcclient.TokenCredential, db.IStandaloneModel, []string, []string) (Usage, error)

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
		return nil, err
	}
	return man.FetchByIdOrName(userCred, id)
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
		projectName := json.GetAnyString(getQuery(r), []string{"project", "tenant"})
		if projectName != "" {
			userCred, err = generateProjectUserCred(ctx, userCred, projectName)
			if err != nil {
				httperrors.GeneralServerError(w, err)
				return
			}
		}
		query := getQuery(r)
		hostTypes := json.GetQueryStringArray(query, "host_type")
		// resourceTypes := json.GetQueryStringArray(query, "resource_type")
		providers := json.GetQueryStringArray(query, "provider")
		usage, err := reporter(userCred, obj, hostTypes, providers)
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
		response(w, usage)
	}
}

func generateProjectUserCred(ctx context.Context, userCred mcclient.TokenCredential, projectName string) (mcclient.TokenCredential, error) {
	project, err := db.TenantCacheManager.FetchTenantByIdOrName(ctx, projectName)
	if err != nil {
		return nil, err
	}
	return &mcclient.SSimpleToken{
		Domain:    project.Domain,
		DomainId:  project.DomainId,
		Project:   project.Name,
		ProjectId: project.Id,
	}, nil
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

func ReportHostUsage(userCred mcclient.TokenCredential, host db.IStandaloneModel, hostTypes []string, providers []string) (Usage, error) {
	return ReportGeneralUsage(userCred, host, hostTypes, providers)
}

func ReportWireUsage(userCred mcclient.TokenCredential, wire db.IStandaloneModel, hostTypes []string, providers []string) (Usage, error) {
	return ReportGeneralUsage(userCred, wire, hostTypes, providers)
}

func ReportCloudAccountUsage(userCred mcclient.TokenCredential, account db.IStandaloneModel, hostTypes []string, providers []string) (Usage, error) {
	return ReportGeneralUsage(userCred, account, hostTypes, providers)
}

func ReportCloudProviderUsage(userCred mcclient.TokenCredential, provider db.IStandaloneModel, hostTypes []string, providers []string) (Usage, error) {
	return ReportGeneralUsage(userCred, provider, hostTypes, providers)
}

func ReportSchedtagUsage(userCred mcclient.TokenCredential, schedtag db.IStandaloneModel, hostTypes []string, providers []string) (Usage, error) {
	return ReportGeneralUsage(userCred, schedtag, hostTypes, providers)
}

func ReportZoneUsage(userCred mcclient.TokenCredential, zone db.IStandaloneModel, hostTypes []string, providers []string) (Usage, error) {
	return ReportGeneralUsage(userCred, zone, hostTypes, providers)
}

func ReportCloudRegionUsage(userCred mcclient.TokenCredential, cloudRegion db.IStandaloneModel, hostTypes []string, providers []string) (Usage, error) {
	return ReportGeneralUsage(userCred, cloudRegion, hostTypes, providers)
}

func getAdminGeneralUsage(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string, providers []string) (count Usage, err error) {
	count = RegionUsage(providers)
	zone := ZoneUsage(providers)
	count.Include(zone)
	vpc := VpcUsage(providers)
	count.Include(vpc)

	var pmemTotal float64
	var pcpuTotal float64

	hostEnabledUsage := HostEnabledUsage("", userCred, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers)
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

	guestRunningUsage := GuestRunningUsage("all.running_servers", nil, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers)
	runningMem := guestRunningUsage.Get("all.running_servers.memory").(int)
	runningCpu := guestRunningUsage.Get("all.running_servers.cpu").(int)

	containerRunningUsage := containerUsage("all.containers", nil, rangeObj, hostTypes, nil, providers)
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

	storageUsage := StorageUsage("", rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers)

	count.Include(
		HostAllUsage("", userCred, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers),
		HostAllUsage("prepaid_pool", userCred, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		HostAllUsage("any_pool", userCred, rangeObj, hostTypes, nil, providers),

		hostEnabledUsage,
		HostEnabledUsage("prepaid_pool", userCred, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		HostEnabledUsage("any_pool", userCred, rangeObj, hostTypes, nil, providers),

		BaremetalUsage(userCred, rangeObj, hostTypes, providers),

		storageUsage,
		StorageUsage("prepaid_pool", rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		StorageUsage("any_pool", rangeObj, hostTypes, nil, providers),

		GuestNormalUsage("all.servers", nil, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers),
		GuestNormalUsage("all.servers.prepaid_pool", nil, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		GuestNormalUsage("all.servers.any_pool", nil, rangeObj, hostTypes, nil, providers),

		GuestPendingDeleteUsage("all.pending_delete_servers", nil, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers),
		GuestPendingDeleteUsage("all.pending_delete_servers.prepaid_pool", nil, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		GuestPendingDeleteUsage("all.pending_delete_servers.any_pool", nil, rangeObj, hostTypes, nil, providers),

		GuestReadyUsage("all.ready_servers", nil, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers),
		GuestReadyUsage("all.ready_servers.prepaid_pool", nil, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		GuestReadyUsage("all.ready_servers.any_pool", nil, rangeObj, hostTypes, nil, providers),

		guestRunningUsage,
		GuestRunningUsage("all.running_servers.prepaid_pool", nil, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		GuestRunningUsage("all.running_servers.any_pool", nil, rangeObj, hostTypes, nil, providers),

		containerRunningUsage,

		IsolatedDeviceUsage("", rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers),
		IsolatedDeviceUsage("prepaid_pool", rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		IsolatedDeviceUsage("any_pool", rangeObj, hostTypes, nil, providers),

		WireUsage(rangeObj, hostTypes, providers),

		EipUsage("", rangeObj, providers),

		SnapshotUsage("", rangeObj, providers),
	)

	return
}

func getCommonGeneralUsage(cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string, providers []string) (count Usage, err error) {
	guestNormalUsage := GuestNormalUsage("servers", cred, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers)

	containerUsage := containerUsage("containers", cred, rangeObj, hostTypes, nil, providers)

	eipUsage := EipUsage(cred.GetProjectId(), rangeObj, providers)

	snapshotUsage := SnapshotUsage(cred.GetProjectId(), rangeObj, providers)

	count = guestNormalUsage.Include(
		GuestNormalUsage("servers.prepaid_pool", cred, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		GuestNormalUsage("servers.any_pool", cred, rangeObj, hostTypes, nil, providers),

		GuestRunningUsage("running_servers", cred, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers),
		GuestRunningUsage("running_servers.prepaid_pool", cred, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		GuestRunningUsage("running_servers.any_pool", cred, rangeObj, hostTypes, nil, providers),

		GuestPendingDeleteUsage("pending_delete_servers", cred, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers),
		GuestPendingDeleteUsage("pending_delete_servers.prepaid_pool", cred, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		GuestPendingDeleteUsage("pending_delete_servers.any_pool", cred, rangeObj, hostTypes, nil, providers),

		GuestReadyUsage("ready_servers", cred, rangeObj, hostTypes, []string{models.HostResourceTypeShared}, providers),
		GuestReadyUsage("ready_servers.prepaid_pool", cred, rangeObj, hostTypes, []string{models.HostResourceTypePrepaidRecycle}, providers),
		GuestReadyUsage("ready_servers.any_pool", cred, rangeObj, hostTypes, nil, providers),

		containerUsage,

		NetworkUsage(cred, providers, rangeObj),

		eipUsage,

		snapshotUsage,
	)
	return
}

func ReportGeneralUsage(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string, providers []string) (count Usage, err error) {
	count = make(map[string]interface{})

	isAdmin := false

	if consts.IsRbacEnabled() {
		if policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
			"usages", policy.PolicyActionGet) == rbacutils.AdminAllow {
			isAdmin = true
		}
	} else {
		isAdmin = userCred.IsAdminAllow(consts.GetServiceType(), "usages", policy.PolicyActionGet)
	}

	if isAdmin {
		count, err = getAdminGeneralUsage(userCred, rangeObj, hostTypes, providers)
		if err != nil {
			return
		}
	}

	includeCommon := false
	if consts.IsRbacEnabled() {
		if policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
			"usages", policy.PolicyActionGet) == rbacutils.Deny {
			if !isAdmin {
				err = httperrors.NewForbiddenError("not allow to get usages")
				return
			}
		} else {
			includeCommon = true
		}
	} else {
		includeCommon = true
	}

	if includeCommon {
		var commonUsage map[string]interface{}
		commonUsage, err = getCommonGeneralUsage(userCred, rangeObj, hostTypes, providers)
		if err != nil {
			return
		}
		count.Include(commonUsage)
	}
	return
}

func RegionUsage(providers []string) Usage {
	q := models.CloudregionManager.Query()
	if len(providers) > 0 {
		q = q.In("provider", providers)
	}
	count := make(map[string]interface{})
	count["regions"] = q.Count()
	return count
}

func ZoneUsage(providers []string) Usage {
	regions := models.CloudregionManager.Query().SubQuery()
	subq := regions.Query(regions.Field("id"))
	if len(providers) > 0 {
		subq = subq.In("provider", providers)
	}
	q := models.ZoneManager.Query().In("cloudregion_id", subq.SubQuery())
	count := make(map[string]interface{})
	count["zones"] = q.Count()
	return count
}

func VpcUsage(providers []string) Usage {
	regions := models.CloudregionManager.Query().SubQuery()
	subq := regions.Query(regions.Field("id"))
	if len(providers) > 0 {
		subq = subq.In("provider", providers)
	}
	q := models.VpcManager.Query().In("cloudregion_id", subq.SubQuery())
	count := make(map[string]interface{})
	count["vpcs"] = q.Count()
	return count
}

func StorageUsage(prefix string, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string) Usage {
	sPrefix := "storages"
	dPrefix := "all.disks"
	if len(prefix) > 0 {
		sPrefix = fmt.Sprintf("%s.%s", sPrefix, prefix)
		dPrefix = fmt.Sprintf("%s.%s", dPrefix, prefix)
	}
	count := make(map[string]interface{})
	result := models.StorageManager.TotalCapacity(rangeObj, hostTypes, resourceTypes, providers)
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

func WireUsage(rangeObj db.IStandaloneModel, hostTypes []string, providers []string) Usage {
	count := make(map[string]interface{})
	result := models.WireManager.TotalCount(rangeObj, hostTypes, providers)
	count["wires"] = result.WiresCount
	count["networks"] = result.NetCount
	count["all.nics.guest"] = result.GuestNicCount
	count["all.nics.host"] = result.HostNicCount
	count["all.nics.reserve"] = result.ReservedCount
	return count
}

func NetworkUsage(userCred mcclient.TokenCredential, providers []string, rangeObj db.IStandaloneModel) Usage {
	count := make(map[string]interface{})
	ret := models.NetworkManager.TotalPortCount(userCred, providers, rangeObj)
	count["ports"] = ret.Count
	count["ports_exit"] = ret.CountExt
	return count
}

func HostAllUsage(pref string, userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string) Usage {
	prefix := "hosts"
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	return hostUsage(userCred, prefix, rangeObj, hostTypes, resourceTypes, providers, tristate.None, tristate.None)
}

func HostEnabledUsage(pref string, userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string) Usage {
	prefix := "enabled_hosts"
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	return hostUsage(userCred, prefix, rangeObj, hostTypes, resourceTypes, providers, tristate.True, tristate.None)
}

func BaremetalUsage(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel,
	hostTypes []string, providers []string) Usage {
	prefix := "baremetals"
	count := hostUsage(userCred, prefix, rangeObj, hostTypes, nil, providers, tristate.None, tristate.True)
	delete(count, fmt.Sprintf("%s.memory.virtual", prefix))
	delete(count, fmt.Sprintf("%s.cpu.virtual", prefix))
	return count
}

func hostUsage(
	userCred mcclient.TokenCredential, prefix string,
	rangeObj db.IStandaloneModel, hostTypes []string,
	resourceTypes []string, providers []string,
	enabled, isBaremetal tristate.TriState,
) Usage {
	count := make(map[string]interface{})

	result := models.HostManager.TotalCount(userCred, rangeObj, "", "", hostTypes, resourceTypes, providers, enabled, isBaremetal)
	count[prefix] = result.Count
	count[fmt.Sprintf("%s.memory", prefix)] = result.Memory
	count[fmt.Sprintf("%s.cpu", prefix)] = result.CPU
	count[fmt.Sprintf("%s.memory.virtual", prefix)] = result.MemoryVirtual
	count[fmt.Sprintf("%s.cpu.virtual", prefix)] = result.CPUVirtual

	return count
}

func GuestNormalUsage(prefix string, cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string) Usage {
	return guestUsage(prefix, cred, rangeObj, hostTypes, resourceTypes, providers, nil, false)
}

func GuestPendingDeleteUsage(prefix string, cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string) Usage {
	return guestUsage(prefix, cred, rangeObj, hostTypes, resourceTypes, providers, nil, true)
}

func GuestRunningUsage(prefix string, cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string) Usage {
	return guestUsage(prefix, cred, rangeObj, hostTypes, resourceTypes, providers, []string{models.VM_RUNNING}, false)
}

func GuestReadyUsage(prefix string, cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string) Usage {
	return guestUsage(prefix, cred, rangeObj, hostTypes, resourceTypes, providers, []string{models.VM_READY}, false)
}

func guestHypervisorsUsage(
	prefix string,
	userCred mcclient.TokenCredential,
	rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string,
	status, hypervisors []string,
	pendingDelete bool,
) Usage {
	projectId := ""
	if userCred != nil {
		projectId = userCred.GetProjectId()
	}
	guest := models.GuestManager.TotalCount(projectId, rangeObj, status, hypervisors, true, pendingDelete, hostTypes, resourceTypes, providers)
	count := make(map[string]interface{})
	count[prefix] = guest.TotalGuestCount
	count[fmt.Sprintf("%s.cpu", prefix)] = guest.TotalCpuCount
	count[fmt.Sprintf("%s.memory", prefix)] = guest.TotalMemSize

	if len(hypervisors) == 1 && hypervisors[0] == models.HYPERVISOR_CONTAINER {
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

func guestUsage(prefix string, userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string,
	status []string, pendingDelete bool) Usage {
	hypervisors := sets.NewString(models.HYPERVISORS...)
	hypervisors.Delete(models.HYPERVISOR_CONTAINER)
	return guestHypervisorsUsage(prefix, userCred, rangeObj, hostTypes, resourceTypes, providers, status, hypervisors.List(), pendingDelete)
}

func containerUsage(prefix string, userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel,
	hostTypes []string, resourceTypes []string, providers []string) Usage {
	hypervisors := []string{models.HYPERVISOR_CONTAINER}
	return guestHypervisorsUsage(prefix, userCred, rangeObj, hostTypes, resourceTypes, providers, nil, hypervisors, false)
}

func IsolatedDeviceUsage(pref string, rangeObj db.IStandaloneModel, hostType []string, resourceTypes []string, providers []string) Usage {
	prefix := "isolated_devices"
	if len(pref) > 0 {
		prefix = fmt.Sprintf("%s.%s", prefix, pref)
	}
	ret := models.IsolatedDeviceManager.TotalCount(hostType, resourceTypes, providers, rangeObj)
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

func EipUsage(projectId string, rangeObj db.IStandaloneModel, providers []string) Usage {
	eipUsage := models.ElasticipManager.TotalCount(projectId, rangeObj, providers)
	count := make(map[string]interface{})
	count[getKey(projectId, "eip")] = eipUsage.Total()
	count[getKey(projectId, "eip.public_ip")] = eipUsage.PublicIPCount
	count[getKey(projectId, "eip.floating_ip")] = eipUsage.EIPCount
	count[getKey(projectId, "eip.floating_ip.used")] = eipUsage.EIPUsedCount
	return count
}

func SnapshotUsage(projectId string, rangeObj db.IStandaloneModel, providers []string) Usage {
	cnt := models.TotalSnapshotCount(projectId, rangeObj, providers)
	count := make(map[string]interface{})
	count[getKey(projectId, "snapshot")] = cnt
	return count
}
