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
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
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

type objUsageFunc func(mcclient.TokenCredential, db.IStandaloneModel, []string) (Usage, error)

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
		userCred := auth.FetchUserCredential(ctx)
		obj, err := getRangeObj(ctx, manager, userCred)
		if err != nil {
			httperrors.NotFoundError(w, err.Error())
			return
		}
		usage, err := reporter(userCred, obj, getHostTypes(r))
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
		"":            rangeObjHandler(nil, ReportGeneralUsage),
		"zone":        rangeObjHandler(models.ZoneManager, ReportZoneUsage),
		"wire":        rangeObjHandler(models.WireManager, ReportWireUsage),
		"schedtag":    rangeObjHandler(models.SchedtagManager, ReportSchedtagUsage),
		"host":        rangeObjHandler(models.HostManager, ReportHostUsage),
		"vcenter":     rangeObjHandler(models.VCenterManager, ReportVCenterUsage),
		"cloudregion": rangeObjHandler(models.CloudregionManager, ReportCloudRegionUsage),
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

func getHostTypes(r *http.Request) []string {
	types := make([]string, 0)
	t, _ := getQuery(r).GetString("host_type")
	if len(t) != 0 {
		types = append(types, t)
	}
	return types
}

func ReportHostUsage(userCred mcclient.TokenCredential, host db.IStandaloneModel, hostTypes []string) (Usage, error) {
	return ReportGeneralUsage(userCred, host, hostTypes)
}

func ReportWireUsage(userCred mcclient.TokenCredential, wire db.IStandaloneModel, hostTypes []string) (Usage, error) {
	return ReportGeneralUsage(userCred, wire, hostTypes)
}

func ReportVCenterUsage(userCred mcclient.TokenCredential, vcenter db.IStandaloneModel, hostTypes []string) (Usage, error) {
	return ReportGeneralUsage(userCred, vcenter, hostTypes)
}

func ReportSchedtagUsage(userCred mcclient.TokenCredential, schedtag db.IStandaloneModel, hostTypes []string) (Usage, error) {
	return ReportGeneralUsage(userCred, schedtag, hostTypes)
}

func ReportZoneUsage(userCred mcclient.TokenCredential, zone db.IStandaloneModel, hostTypes []string) (Usage, error) {
	return ReportGeneralUsage(userCred, zone, hostTypes)
}

func ReportCloudRegionUsage(userCred mcclient.TokenCredential, cloudRegion db.IStandaloneModel, hostTypes []string) (Usage, error) {
	return ReportGeneralUsage(userCred, cloudRegion, hostTypes)
}

//func ReportGuestUsage()

func getAdminGeneralUsage(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) (count Usage, err error) {
	count = ZoneUsage()
	var pmemTotal float64
	var pcpuTotal float64

	hostEnabledUsage := HostEnabledUsage(userCred, rangeObj, hostTypes)
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
	guestRunningUsage := GuestRunningUsage("all.running_servers", nil, rangeObj, hostTypes)
	runningMem := guestRunningUsage.Get("all.running_servers.memory").(int)
	runningCpu := guestRunningUsage.Get("all.running_servers.cpu").(int)
	containerRunningUsage := containerUsage("all.containers", nil, rangeObj)
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

	storageUsage := StorageUsage(rangeObj, hostTypes)
	storageCmtRate := 0.0
	disksSize := storageUsage.Get("all.disks").(int64)
	storageSize := storageUsage.Get("storages").(int64)
	if storageSize > 0 {
		storageCmtRate = utils.FloatRound(float64(disksSize)/float64(storageSize), 2)
	}
	count.Add("storages_commit_rate", storageCmtRate)

	count.Include(
		HostAllUsage(userCred, rangeObj, hostTypes),
		hostEnabledUsage,
		BaremetalUsage(userCred, rangeObj, hostTypes),
		storageUsage,
		GuestNormalUsage("all.servers", nil, rangeObj, hostTypes),
		GuestPendingDeleteUsage("all.pending_delete_servers", nil, rangeObj, hostTypes),
		GuestReadyUsage("all.ready_servers", nil, rangeObj, hostTypes),
		guestRunningUsage,
		containerRunningUsage,
		IsolatedDeviceUsage(rangeObj, hostTypes),
		WireUsage(rangeObj, hostTypes),
		NetworkUsage(userCred, rangeObj),
		EipUsage(rangeObj, hostTypes),
	)

	return
}

func getCommonGeneralUsage(cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) (count Usage, err error) {
	guestNormalUsage := GuestNormalUsage("servers", cred, rangeObj, hostTypes)
	guestRunningUsage := GuestRunningUsage("running_servers", cred, rangeObj, hostTypes)
	guestPendingDeleteUsage := GuestPendingDeleteUsage("pending_delete_servers", cred, rangeObj, hostTypes)
	guestReadyUsage := GuestReadyUsage("ready_servers", cred, rangeObj, hostTypes)
	containerUsage := containerUsage("containers", cred, rangeObj)
	count = guestNormalUsage.Include(
		guestRunningUsage,
		guestPendingDeleteUsage,
		guestReadyUsage,
		containerUsage,
	)
	return
}

func ReportGeneralUsage(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) (count Usage, err error) {
	count = make(map[string]interface{})

	isAdmin := false

	if db.IsGlobalRbacEnabled() {
		if db.PolicyManager.Allow(true, userCred, db.GetGlobalServiceType(),
			"usages", db.PolicyActionGet) {
			isAdmin = true
		}
	} else {
		isAdmin = userCred.IsSystemAdmin()
	}

	if isAdmin {
		count, err = getAdminGeneralUsage(userCred, rangeObj, hostTypes)
		if err != nil {
			return
		}
	}

	if db.IsGlobalRbacEnabled() {
		if ! db.PolicyManager.Allow(false, userCred, db.GetGlobalServiceType(),
			"usages", db.PolicyActionGet) {
			err = httperrors.NewForbiddenError("not allow to get usages")
			return
		}
	}

	commonUsage, err := getCommonGeneralUsage(userCred, rangeObj, hostTypes)
	if err != nil {
		return
	}
	count.Include(commonUsage)
	return
}

func ZoneUsage() Usage {
	count := make(map[string]interface{})
	count["zones"] = models.ZoneManager.Count()
	return count
}

func StorageUsage(rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	sPrefix := "storages"
	dPrefix := "all.disks"
	count := make(map[string]interface{})
	result := models.StorageManager.TotalCapacity(rangeObj, hostTypes)
	count[sPrefix] = result.Capacity
	count[fmt.Sprintf("%s.virtual", sPrefix)] = result.CapacityVirtual
	count[dPrefix] = result.CapacityUsed
	count[fmt.Sprintf("%s.unready", dPrefix)] = result.CapacityUnread
	count[fmt.Sprintf("%s.attached", dPrefix)] = result.AttachedCapacity
	count[fmt.Sprintf("%s.detached", dPrefix)] = result.DetachedCapacity
	return count
}

func WireUsage(rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	count := make(map[string]interface{})
	result := models.WireManager.TotalCount(rangeObj, hostTypes)
	count["wires"] = result.WiresCount
	count["networks"] = result.NetCount
	count["all.nics.guest"] = result.GuestNicCount
	count["all.nics.host"] = result.HostNicCount
	count["all.nics.reserve"] = result.ReservedCount
	return count
}

func NetworkUsage(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel) Usage {
	count := make(map[string]interface{})
	ret := models.NetworkManager.TotalPortCount(userCred, rangeObj)
	count["ports"] = ret.Count
	count["ports_exit"] = ret.CountExt
	return count
}

func HostAllUsage(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	prefix := "hosts"
	return hostUsage(userCred, prefix, rangeObj, hostTypes, tristate.None, tristate.None)
}

func HostEnabledUsage(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	prefix := "enabled_hosts"
	return hostUsage(userCred, prefix, rangeObj, hostTypes, tristate.True, tristate.None)
}

func BaremetalUsage(userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	prefix := "baremetals"
	count := hostUsage(userCred, prefix, rangeObj, hostTypes, tristate.None, tristate.True)
	delete(count, fmt.Sprintf("%s.memory.virtual", prefix))
	delete(count, fmt.Sprintf("%s.cpu.virtual", prefix))
	return count
}

func hostUsage(
	userCred mcclient.TokenCredential, prefix string,
	rangeObj db.IStandaloneModel, hostTypes []string,
	enabled, isBaremetal tristate.TriState,
) Usage {
	count := make(map[string]interface{})

	result := models.HostManager.TotalCount(userCred, rangeObj, "", "", hostTypes, enabled, isBaremetal)
	count[prefix] = result.Count
	count[fmt.Sprintf("%s.memory", prefix)] = result.Memory
	count[fmt.Sprintf("%s.cpu", prefix)] = result.CPU
	count[fmt.Sprintf("%s.memory.virtual", prefix)] = result.MemoryVirtual
	count[fmt.Sprintf("%s.cpu.virtual", prefix)] = result.CPUVirtual

	return count
}

func GuestNormalUsage(prefix string, cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	return guestUsage(prefix, cred, rangeObj, hostTypes, nil, false)
}

func GuestPendingDeleteUsage(prefix string, cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	return guestUsage(prefix, cred, rangeObj, hostTypes, nil, true)
}

func GuestRunningUsage(prefix string, cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	return guestUsage(prefix, cred, rangeObj, hostTypes, []string{models.VM_RUNNING}, false)
}

func GuestReadyUsage(prefix string, cred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	return guestUsage(prefix, cred, rangeObj, hostTypes, []string{models.VM_READY}, false)
}

func guestHypervisorsUsage(
	prefix string,
	userCred mcclient.TokenCredential,
	rangeObj db.IStandaloneModel,
	hostTypes, status, hypervisors []string,
	pendingDelete bool,
) Usage {
	hostType := ""
	if len(hostTypes) != 0 {
		hostType = hostTypes[0]
	}
	projectId := ""
	if userCred != nil {
		projectId = userCred.GetProjectId()
	}
	guest := models.GuestManager.TotalCount(projectId, rangeObj, status, hypervisors, true, pendingDelete, hostType)
	count := make(map[string]interface{})
	count[prefix] = guest.TotalGuestCount
	count[fmt.Sprintf("%s.cpu", prefix)] = guest.TotalCpuCount
	count[fmt.Sprintf("%s.memory", prefix)] = guest.TotalMemSize

	if len(hypervisors) == 1 && hypervisors[0] == models.HYPERVISOR_CONTAINER {
		return count
	}

	count[fmt.Sprintf("%s.disk", prefix)] = guest.TotalDiskSize
	count[fmt.Sprintf("%s.isolated_devices", prefix)] = guest.TotalIsolatedCount

	return count
}

func guestUsage(prefix string, userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel, hostTypes, status []string, pendingDelete bool) Usage {
	hypervisors := sets.NewString(models.HYPERVISORS...)
	hypervisors.Delete(models.HYPERVISOR_CONTAINER)
	return guestHypervisorsUsage(prefix, userCred, rangeObj, hostTypes, status, hypervisors.List(), pendingDelete)
}

func containerUsage(prefix string, userCred mcclient.TokenCredential, rangeObj db.IStandaloneModel) Usage {
	hypervisors := []string{models.HYPERVISOR_CONTAINER}
	return guestHypervisorsUsage(prefix, userCred, rangeObj, nil, nil, hypervisors, false)
}

func IsolatedDeviceUsage(rangeObj db.IStandaloneModel, hostType []string) Usage {
	prefix := "isolated_devices"
	ret := models.IsolatedDeviceManager.TotalCount(hostType, rangeObj)
	count := make(map[string]interface{})
	count[prefix] = ret.Devices
	return count
}

func EipUsage(rangeObj db.IStandaloneModel, hostTypes []string) Usage {
	eipUsage := models.ElasticipManager.TotalCount("", rangeObj, hostTypes)
	count := make(map[string]interface{})
	count["eip.all"] = eipUsage.Total()
	count["eip.public_ip"] = eipUsage.PublicIPCount
	count["eip.floating_ip"] = eipUsage.EIPCount
	count["eip.floating_ip.used"] = eipUsage.EIPUsedCount
	return count
}
