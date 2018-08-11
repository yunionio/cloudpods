package models

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sysutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	VM_INIT            = "init"
	VM_UNKNOWN         = "unknown"
	VM_SCHEDULE        = "schedule"
	VM_SCHEDULE_FAILED = "sched_fail"
	VM_CREATE_NETWORK  = "network"
	VM_NETWORK_FAILED  = "net_fail"
	VM_DEVICE_FAILED   = "dev_fail"
	VM_CREATE_FAILED   = "create_fail"
	VM_CREATE_DISK     = "disk"
	VM_DISK_FAILED     = "disk_fail"
	VM_START_DEPLOY    = "start_deploy"
	VM_DEPLOYING       = "deploying"
	VM_DEPLOY_FAILED   = "deploy_fail"
	VM_READY           = "ready"
	VM_START_START     = "start_start"
	VM_STARTING        = "starting"
	VM_START_FAILED    = "start_fail" // # = ready
	VM_RUNNING         = "running"
	VM_START_STOP      = "start_stop"
	VM_STOPPING        = "stopping"
	VM_STOP_FAILED     = "stop_fail" // # = running

	VM_START_SUSPEND  = "start_suspend"
	VM_SUSPENDING     = "suspending"
	VM_SUSPEND        = "suspend"
	VM_SUSPEND_FAILED = "suspend_failed"

	VM_START_DELETE = "start_delete"
	VM_DELETE_FAIL  = "delete_fail"
	VM_DELETING     = "deleting"

	VM_START_MIGRATE  = "start_migrate"
	VM_MIGRATING      = "migrating"
	VM_MIGRATE_FAILED = "migrate_failed"

	VM_CHANGE_FLAVOR     = "change_flavor"
	VM_REBUILD_ROOT      = "rebuild_root"
	VM_REBUILD_ROOT_FAIL = "rebld_root_fail"

	VM_START_SNAPSHOT  = "snapshot_start"
	VM_SNAPSHOT        = "snapshot"
	VM_SNAPSHOT_STREAM = "block_stream"
	VM_SNAPSHOT_SUCC   = "snapshot_succ"
	VM_SNAPSHOT_FAILED = "snapshot_failed"

	VM_SYNCING_STATUS = "syncing"
	VM_SYNC_CONFIG    = "sync_config"
	VM_SYNC_FAIL      = "sync_fail"

	VM_RESIZE_DISK      = "resize_disk"
	VM_START_SAVE_DISK  = "start_save_disk"
	VM_SAVE_DISK        = "save_disk"
	VM_SAVE_DISK_FAILED = "save_disk_failed"

	VM_RESTORING_SNAPSHOT = "restoring_snapshot"
	VM_RESTORE_DISK       = "restore_disk"
	VM_RESTORE_STATE      = "restore_state"
	VM_RESTORE_FAILED     = "restore_failed"

	VM_REMOVE_STATEFILE = "remove_state"

	VM_ADMIN = "admin"

	SHUTDOWN_STOP      = "stop"
	SHUTDOWN_TERMINATE = "terminate"

	HYPERVISOR_KVM       = "kvm"
	HYPERVISOR_CONTAINER = "container"
	HYPERVISOR_BAREMETAL = "baremetal"
	HYPERVISOR_ESXI      = "esxi"
	HYPERVISOR_HYPERV    = "hyperv"
	HYPERVISOR_ALIYUN    = "aliyun"

	//	HYPERVISOR_DEFAULT = HYPERVISOR_KVM
	HYPERVISOR_DEFAULT = HYPERVISOR_ALIYUN
)

var VM_RUNNING_STATUS = []string{VM_START_START, VM_STARTING, VM_RUNNING, VM_SNAPSHOT_STREAM}
var VM_CREATING_STATUS = []string{VM_CREATE_NETWORK, VM_CREATE_DISK, VM_START_DEPLOY, VM_DEPLOYING}

var HYPERVISORS = []string{HYPERVISOR_KVM, HYPERVISOR_BAREMETAL, HYPERVISOR_ESXI, HYPERVISOR_CONTAINER, HYPERVISOR_ALIYUN}

// var HYPERVISORS = []string{HYPERVISOR_ALIYUN}

var HYPERVISOR_HOSTTYPE = map[string]string{
	HYPERVISOR_KVM:       HOST_TYPE_HYPERVISOR,
	HYPERVISOR_BAREMETAL: HOST_TYPE_BAREMETAL,
	HYPERVISOR_ESXI:      HOST_TYPE_ESXI,
	HYPERVISOR_CONTAINER: HOST_TYPE_KUBELET,
	HYPERVISOR_ALIYUN:    HOST_TYPE_ALIYUN,
}

var HOSTTYPE_HYPERVISOR = map[string]string{
	HOST_TYPE_HYPERVISOR: HYPERVISOR_KVM,
	HOST_TYPE_BAREMETAL:  HYPERVISOR_BAREMETAL,
	HOST_TYPE_ESXI:       HYPERVISOR_ESXI,
	HOST_TYPE_KUBELET:    HYPERVISOR_CONTAINER,
	HOST_TYPE_ALIYUN:     HYPERVISOR_ALIYUN,
}

type SGuestManager struct {
	db.SVirtualResourceBaseManager
}

var GuestManager *SGuestManager

func init() {
	GuestManager = &SGuestManager{SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SGuest{}, "guests_tbl", "server", "servers")}
	GuestManager.SetAlias("guest", "guests")
}

type SGuest struct {
	db.SVirtualResourceBase

	VcpuCount int8 `nullable:"false" default:"1" list:"user" create:"optional"` // Column(TINYINT, nullable=False, default=1)
	VmemSize  int  `nullable:"false" list:"user" create:"required"`             // Column(Integer, nullable=False)

	BootOrder string `width:"8" charset:"ascii" nullable:"true" default:"cdn" list:"user" update:"user" create:"optional"` // Column(VARCHAR(8, charset='ascii'), nullable=True, default='cdn')

	DisableDelete    tristate.TriState `nullable:"false" default:"true" list:"user" update:"user" create:"optional"`           // Column(Boolean, nullable=False, default=True)
	ShutdownBehavior string            `width:"16" charset:"ascii" default:"stop" list:"user" update:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), default=SHUTDOWN_STOP)

	KeypairId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	HostId string `width:"36" charset:"ascii" nullable:"true" list:"admin" get:"admin"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	Vga     string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Vdi     string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Machine string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Bios    string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	OsType  string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	FlavorId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	SecgrpId      string `width:"36" charset:"ascii" nullable:"true" get:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	AdminSecgrpId string `width:"36" charset:"ascii" nullable:"true" get:"admin"`                  // Column(VARCHAR(36, charset='ascii'), nullable=True)

	Hypervisor string `width:"16" charset:"ascii" nullable:"false" default:"kvm" list:"user" create:"required"` // Column(VARCHAR(16, charset='ascii'), nullable=False, default=HYPERVISOR_DEFAULT)
}

func (manager *SGuestManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	if query.Contains("host") || query.Contains("wire") || query.Contains("zone") {
		if !userCred.IsSystemAdmin() {
			return false
		}
	}
	return manager.SVirtualResourceBaseManager.AllowListItems(ctx, userCred, query)
}

func (manager *SGuestManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	queryDict, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("invalid querystring format")
	}
	isBMstr, _ := queryDict.GetString("baremetal")
	if len(isBMstr) > 0 && utils.ToBool(isBMstr) {
		queryDict.Add(jsonutils.NewString(HYPERVISOR_BAREMETAL), "hypervisor")
		queryDict.Remove("baremetal")
	}
	hypervisor, _ := queryDict.GetString("hypervisor")
	if len(hypervisor) > 0 {
		q = q.Equals("hypervisor", hypervisor)
	}
	hostFilter, _ := queryDict.GetString("host")
	zoneFilter, _ := queryDict.GetString("zone")
	wireFilter, _ := queryDict.GetString("wire")
	networkFilter, _ := queryDict.GetString("network")
	diskFilter, _ := queryDict.GetString("disk")
	var sq *sqlchemy.SSubQuery
	if len(hostFilter) > 0 {
		host, _ := HostManager.FetchByIdOrName("", hostFilter)
		if host == nil {
			return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("host %s not found", hostFilter))
		}
		sq = HostManager.Query("id").Equals("id", host.GetId()).SubQuery()
	} else if len(zoneFilter) > 0 {
		zone, _ := ZoneManager.FetchByIdOrName("", zoneFilter)
		if zone == nil {
			return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("zone %s not found", zoneFilter))
		}
		hostTable := HostManager.Query().SubQuery()
		zoneTable := ZoneManager.Query().SubQuery()
		sq = hostTable.Query(hostTable.Field("id")).Join(zoneTable,
			sqlchemy.Equals(zoneTable.Field("id"), hostTable.Field("zone_id"))).Filter(sqlchemy.Equals(zoneTable.Field("id"), zone.GetId())).SubQuery()
	} else if len(wireFilter) > 0 {
		wire, _ := WireManager.FetchByIdOrName("", wireFilter)
		if wire == nil {
			return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("wire %s not found", wireFilter))
		}
		hostTable := HostManager.Query().SubQuery()
		hostWire := HostwireManager.Query().SubQuery()
		sq = hostTable.Query(hostTable.Field("id")).Join(hostWire, sqlchemy.Equals(hostWire.Field("host_id"), hostTable.Field("id"))).Filter(sqlchemy.Equals(hostWire.Field("wire_id"), wire.GetId())).SubQuery()
	} else if len(networkFilter) > 0 {
		netI, _ := NetworkManager.FetchByIdOrName(userCred.GetProjectId(), networkFilter)
		if netI == nil {
			return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("network %s not found", networkFilter))
		}
		net := netI.(*SNetwork)
		hostTable := HostManager.Query().SubQuery()
		hostWire := HostwireManager.Query().SubQuery()
		sq = hostTable.Query(hostTable.Field("id")).Join(hostWire,
			sqlchemy.Equals(hostWire.Field("host_id"), hostTable.Field("id"))).Filter(sqlchemy.Equals(hostWire.Field("wire_id"), net.WireId)).SubQuery()
	} else if len(diskFilter) > 0 {
		diskI, _ := DiskManager.FetchByIdOrName(userCred.GetProjectId(), diskFilter)
		if diskI == nil {
			return nil, httperrors.NewResourceNotFoundError(fmt.Sprintf("disk %s not found", diskFilter))
		}
		disk := diskI.(*SDisk)
		guestdisks := GuestdiskManager.Query().SubQuery()
		count := guestdisks.Query().Filter(sqlchemy.AND(
			sqlchemy.Equals(guestdisks.Field("disk_id"), disk.Id),
			sqlchemy.IsFalse(guestdisks.Field("deleted")))).Count()
		if count > 0 {
			sgq := guestdisks.Query(guestdisks.Field("guest_id")).
				Filter(sqlchemy.AND(
					sqlchemy.Equals(guestdisks.Field("disk_id"), disk.Id),
					sqlchemy.IsFalse(guestdisks.Field("deleted"))))
			q = q.Filter(sqlchemy.In(q.Field("id"), sgq))
		} else {
			hosts := HostManager.Query().SubQuery()
			hoststorages := HoststorageManager.Query().SubQuery()
			storages := StorageManager.Query().SubQuery()
			sq = hosts.Query(hosts.Field("id")).
				Join(hoststorages, sqlchemy.AND(
					sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")),
					sqlchemy.IsFalse(hoststorages.Field("deleted")))).
				Join(storages, sqlchemy.AND(
					sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")),
					sqlchemy.IsFalse(storages.Field("deleted")))).
				Filter(sqlchemy.Equals(storages.Field("id"), disk.StorageId)).SubQuery()
		}
	}
	if sq != nil {
		q = q.In("host_id", sq)
	}
	gpu, _ := queryDict.GetString("gpu")
	if len(gpu) != 0 {
		isodev := IsolatedDeviceManager.Query().SubQuery()
		sgq := isodev.Query(isodev.Field("guest_id")).
			Filter(sqlchemy.AND(
				sqlchemy.IsNotNull(isodev.Field("guest_id")),
				sqlchemy.Startswith(isodev.Field("dev_type"), "GPU")))
		showGpu := utils.ToBool(gpu)
		cond := sqlchemy.NotIn
		if showGpu {
			cond = sqlchemy.In
		}
		q = q.Filter(cond(q.Field("id"), sgq))
	}
	return q, nil
}

func (manager *SGuestManager) ExtraSearchConditions(ctx context.Context, q *sqlchemy.SQuery, like string) []sqlchemy.ICondition {
	var sq *sqlchemy.SSubQuery
	if regutils.MatchIP4Addr(like) {
		sq = GuestnetworkManager.Query("guest_id").Equals("ip_addr", like).SubQuery()
	} else if regutils.MatchMacAddr(like) {
		sq = GuestnetworkManager.Query("guest_id").Equals("mac_addr", like).SubQuery()
	}
	if sq != nil {
		return []sqlchemy.ICondition{sqlchemy.In(q.Field("id"), sq)}
	}
	return nil
}

func (guest *SGuest) GetHypervisor() string {
	if len(guest.Hypervisor) == 0 {
		return HYPERVISOR_DEFAULT
	} else {
		return guest.Hypervisor
	}
}

func (guest *SGuest) GetHostType() string {
	return HYPERVISOR_HOSTTYPE[guest.Hypervisor]
}

func (guest *SGuest) GetDriver() IGuestDriver {
	hypervisor := guest.GetHypervisor()
	if !utils.IsInStringArray(hypervisor, HYPERVISORS) {
		log.Fatalf("Unsupported hypervisor %s", hypervisor)
	}
	return GetDriver(hypervisor)
}

func (guest *SGuest) ValidateDeleteCondition(ctx context.Context) error {
	if guest.DisableDelete.IsTrue() {
		return fmt.Errorf("Virtual server is locked, cannot delete")
	}
	return guest.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (guest *SGuest) GetDisksQuery() *sqlchemy.SQuery {
	return GuestdiskManager.Query().Equals("guest_id", guest.Id)
}

func (guest *SGuest) DiskCount() int {
	return guest.GetDisksQuery().Count()
}

func (guest *SGuest) GetDisks() []SGuestdisk {
	disks := make([]SGuestdisk, 0)
	q := guest.GetDisksQuery().Asc("index")
	err := db.FetchModelObjects(GuestdiskManager, q, &disks)
	if err != nil {
		log.Errorf("Getdisks error: %s", err)
	}
	return disks
}

func (guest *SGuest) GetGuestDisk(diskId string) *SGuestdisk {
	guestdisk, err := db.NewModelObject(GuestdiskManager)
	if err != nil {
		log.Errorf("new guestdisk model failed: %s", err)
		return nil
	}
	q := guest.GetDisksQuery()
	err = q.Equals("disk_id", diskId).First(guestdisk)
	if err != nil {
		log.Errorf("GetGuestDisk error: %s", err)
		return nil
	}
	return guestdisk.(*SGuestdisk)
}

func (guest *SGuest) GetNetworksQuery() *sqlchemy.SQuery {
	return GuestnetworkManager.Query().Equals("guest_id", guest.Id)
}

func (guest *SGuest) NetworkCount() int {
	return guest.GetNetworksQuery().Count()
}

func (guest *SGuest) GetNetworks() []SGuestnetwork {
	guestnics := make([]SGuestnetwork, 0)
	q := guest.GetNetworksQuery().Asc("index")
	err := db.FetchModelObjects(GuestnetworkManager, q, &guestnics)
	if err != nil {
		log.Errorf("GetNetworks error: %s", err)
	}
	return guestnics
}

func (guest *SGuest) IsNetworkAllocated() bool {
	guestnics := guest.GetNetworks()
	for _, gn := range guestnics {
		if !gn.IsAllocated() {
			return false
		}
	}
	return true
}

func (guest *SGuest) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	guest.HostId = ""
	return guest.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
}

func (guest *SGuest) GetHost() *SHost {
	if len(guest.HostId) > 0 && regutils.MatchUUID(guest.HostId) {
		host, _ := HostManager.FetchById(guest.HostId)
		return host.(*SHost)
	}
	return nil
}

func (guest *SGuest) SetHostId(hostId string) error {
	_, err := guest.GetModelManager().TableSpec().Update(guest, func() error {
		guest.HostId = hostId
		return nil
	})
	return err
}

func validateMemCpuData(data jsonutils.JSONObject) (int, int, error) {
	vmemSize := 0
	vcpuCount := 0
	var err error

	hypervisor, _ := data.GetString("hypervisor")
	if len(hypervisor) == 0 {
		hypervisor = HYPERVISOR_DEFAULT
	}
	driver := GetDriver(hypervisor)

	vmemStr, _ := data.GetString("vmem_size")
	if len(vmemStr) > 0 {
		if !regutils.MatchSize(vmemStr) {
			return 0, 0, httperrors.NewInputParameterError("Memory size must be number[+unit], like 256M, 1G or 256")
		}
		vmemSize, err = fileutils.GetSizeMb(vmemStr, 'M', 1024)
		if err != nil {
			return 0, 0, err
		}
		maxVmemGb := driver.GetMaxVMemSizeGB()
		if vmemSize < 64 || vmemSize > maxVmemGb*1024 {
			return 0, 0, httperrors.NewInputParameterError("Memory size must be 64MB ~ %d GB", maxVmemGb)
		}
	}
	vcpuStr, _ := data.GetString("vcpu_count")
	if len(vcpuStr) > 0 {
		if !regutils.MatchInteger(vcpuStr) {
			return 0, 0, httperrors.NewInputParameterError("CPU core count must be integer")
		}
		vcpuCount, _ = strconv.Atoi(vcpuStr)
		maxVcpuCount := driver.GetMaxVCpuCount()
		if vcpuCount < 1 || vcpuCount > maxVcpuCount {
			return 0, 0, httperrors.NewInputParameterError("CPU core count must be 1 ~ %d", maxVcpuCount)
		}
	}
	return vmemSize, vcpuCount, nil
}

func (self *SGuest) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	vmemSize, vcpuCount, err := validateMemCpuData(data)
	if err != nil {
		return nil, err
	}
	if vmemSize > 0 {
		data.Add(jsonutils.NewInt(int64(vmemSize)), "vmem_size")
	}
	if vcpuCount > 0 {
		data.Add(jsonutils.NewInt(int64(vcpuCount)), "vcpu_count")
	}

	err = self.checkUpdateQuota(ctx, userCred, vcpuCount, vmemSize)
	if err != nil {
		return nil, err
	}

	// if data.Contains("name") {
	//	return nil, httperrors.NewInputParameterError("cannot update server name")
	// }
	/* if self.GetHypervisor() == HYPERVISOR_BAREMETAL {
		return nil, httperrors.NewInputParameterError("Cannot modify memory for baremetal")
	}
	if ! utils.IsInStringArray(self.Status, []string {VM_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot modify Memory and CPU in status %s", self.Status)
	}*/
	// return nil, httperrors.NewInputParameterError("cannot update guest vmem_size")
	//}
	return self.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SGuestManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	vmemSize, vcpuCount, err := validateMemCpuData(data)
	if err != nil {
		return nil, err
	}
	if vmemSize == 0 {
		return nil, httperrors.NewInputParameterError("Missing memory size")
	}
	if vcpuCount == 0 {
		vcpuCount = 1
	}
	data.Add(jsonutils.NewInt(int64(vmemSize)), "vmem_size")
	data.Add(jsonutils.NewInt(int64(vcpuCount)), "vcpu_count")

	disk0Json, _ := data.Get("disk.0")
	if disk0Json == nil {
		return nil, httperrors.NewInputParameterError("No disk information provided")
	}
	diskConfig, err := parseDiskInfo(ctx, userCred, disk0Json)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Invalid root image: %s", err)
	}

	data.Add(jsonutils.Marshal(diskConfig), "disk.0")

	imgProperties := diskConfig.ImageProperties
	if imgProperties == nil || len(imgProperties) == 0 {
		imgProperties = map[string]string{"os_type": "Linux"}
	}

	hypervisor, _ := data.GetString("hypervisor")
	osType, _ := data.GetString("os_type")

	osProf, err := osprofile.GetOSProfileFromImageProperties(imgProperties, hypervisor)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Invalid root image: %s", err)
	}

	if len(osProf.Hypervisor) > 0 && len(hypervisor) == 0 {
		hypervisor = osProf.Hypervisor
		data.Add(jsonutils.NewString(osProf.Hypervisor), "hypervisor")
	}
	if len(osProf.OSType) > 0 && len(osType) == 0 {
		osType = osProf.OSType
		data.Add(jsonutils.NewString(osProf.OSType), "os_type")
	}
	data.Add(jsonutils.Marshal(osProf), "__os_profile__")

	if jsonutils.QueryBoolean(data, "baremetal", false) {
		hypervisor = HYPERVISOR_BAREMETAL
	}

	if data.Contains("prefer_baremetal") || data.Contains("prefer_host") {
		if !userCred.IsSystemAdmin() {
			return nil, httperrors.NewNotSufficientPrivilegeError("Only system admin can specify preferred host")
		}
		bmName, _ := data.GetString("prefer_host")
		if len(bmName) == 0 {
			bmName, _ = data.GetString("prefer_baremetal")
		}
		bmObj, err := HostManager.FetchByIdOrName("", bmName)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("Host %s not found", bmName)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		baremetal := bmObj.(*SHost)
		if !baremetal.Enabled {
			return nil, httperrors.NewInvalidStatusError("Baremetal %s not enabled", bmName)
		}

		if len(hypervisor) > 0 && hypervisor != HOSTTYPE_HYPERVISOR[baremetal.HostType] {
			return nil, httperrors.NewInputParameterError("cannot run hypervisor %s on specified host with type %s", hypervisor, baremetal.HostType)
		}

		if len(hypervisor) == 0 {
			hypervisor = HOSTTYPE_HYPERVISOR[baremetal.HostType]
		}

		if len(hypervisor) == 0 {
			hypervisor = HYPERVISOR_DEFAULT
		}

		data, err = GetDriver(hypervisor).ValidateCreateHostData(ctx, userCred, bmName, baremetal, data)
		if err != nil {
			return nil, err
		}

	} else {
		schedtags := make(map[string]string)
		if data.Contains("aggregate_strategy") {
			err = data.Unmarshal(&schedtags, "aggregate_strategy")
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid aggregate_strategy")
			}
		}
		for idx := 0; data.Contains(fmt.Sprintf("srvtag.%d", idx)); idx += 1 {
			aggStr, _ := data.GetString(fmt.Sprintf("srvtag.%d", idx))
			if len(aggStr) > 0 {
				parts := strings.Split(aggStr, ":")
				if len(parts) >= 2 && len(parts) > 0 && len(parts[1]) > 0 {
					schedtags[parts[0]] = parts[1]
				}
			}
		}
		if len(schedtags) > 0 {
			schedtags, err = SchedtagManager.ValidateSchedtags(userCred, schedtags)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid aggregate_strategy: %s", err)
			}
			data.Add(jsonutils.Marshal(schedtags), "aggregate_strategy")
		}

		if data.Contains("prefer_wire") {
			wireStr, _ := data.GetString("prefer_wire")
			wireObj, err := WireManager.FetchById(wireStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Wire %s not found", wireStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			wire := wireObj.(*SWire)
			data.Add(jsonutils.NewString(wire.Id), "prefer_wire_id")
			zone := wire.GetZone()
			data.Add(jsonutils.NewString(zone.Id), "prefer_zone_id")
		} else if data.Contains("prefer_zone") {
			zoneStr, _ := data.GetString("prefer_zone")
			zoneObj, err := ZoneManager.FetchById(zoneStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError("Zone %s not found", zoneStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}
			zone := zoneObj.(*SZone)
			data.Add(jsonutils.NewString(zone.Id), "prefer_zone_id")
		}
	}

	if !utils.IsInStringArray(hypervisor, HYPERVISORS) {
		return nil, httperrors.NewInputParameterError("Hypervisor %s not supported", hypervisor)
	}

	data.Add(jsonutils.NewString(hypervisor), "hypervisor")

	for idx := 1; data.Contains(fmt.Sprintf("disk.%d", idx)); idx += 1 {
		diskJson, err := data.Get(fmt.Sprintf("disk.%d", idx))
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid disk description %s", err)
		}
		diskConfig, err := parseDiskInfo(ctx, userCred, diskJson)
		if err != nil {
			return nil, httperrors.NewInputParameterError("parse disk description error %s", err)
		}
		if len(diskConfig.Driver) == 0 {
			diskConfig.Driver = osProf.DiskDriver
		}
		data.Add(jsonutils.Marshal(diskConfig), fmt.Sprintf("disk.%d", idx))
	}

	for idx := 0; data.Contains(fmt.Sprintf("net.%d", idx)); idx += 1 {
		netJson, err := data.Get(fmt.Sprintf("net.%d", idx))
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid network description %s", err)
		}
		netConfig, err := parseNetworkInfo(userCred, netJson)
		if err != nil {
			return nil, httperrors.NewInputParameterError("parse network description error %s", err)
		}
		err = isValidNetworkInfo(userCred, netConfig)
		if err != nil {
			return nil, err
		}
		if len(netConfig.Driver) == 0 {
			netConfig.Driver = osProf.NetDriver
		}
		data.Add(jsonutils.Marshal(netConfig), fmt.Sprintf("net.%d", idx))
	}

	for idx := 0; data.Contains(fmt.Sprintf("isolated_device.%d", idx)); idx += 1 {
		devJson, err := data.Get(fmt.Sprintf("isolated_device.%d", idx))
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid isolated device description %s", err)
		}
		devConfig, err := IsolatedDeviceManager.parseDeviceInfo(userCred, devJson)
		if err != nil {
			return nil, httperrors.NewInputParameterError("parse isolated device description error %s", err)
		}
		err = IsolatedDeviceManager.isValidDeviceinfo(devConfig)
		if err != nil {
			return nil, err
		}
		data.Add(jsonutils.Marshal(devConfig), fmt.Sprintf("isolated_device.%d", idx))
	}

	if data.Contains("cdrom") {
		cdromStr, err := data.GetString("cdrom")
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid cdrom device description %s", err)
		}
		cdromId, err := parseIsoInfo(ctx, userCred, cdromStr)
		if err != nil {
			return nil, httperrors.NewInputParameterError("parse cdrom device info error %s", err)
		}
		data.Add(jsonutils.NewString(cdromId), "cdrom")
	}

	keypairId, _ := data.GetString("keypair")
	if len(keypairId) == 0 {
		keypairId, _ = data.GetString("keypair_id")
	}
	if len(keypairId) > 0 {
		keypairObj, err := KeypairManager.FetchByIdOrName(userCred.GetUserId(), keypairId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Keypair %s not found", keypairId)
		}
		data.Add(jsonutils.NewString(keypairObj.GetId()), "keypair_id")
	} else {
		data.Add(jsonutils.NewString("None"), "keypair_id")
	}

	if data.Contains("secgroup") {
		secGrpId, _ := data.GetString("secgroup")
		secGrpObj, err := SecurityGroupManager.FetchByIdOrName(userCred.GetProjectId(), secGrpId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Secgroup %s not found", secGrpId)
		}
		data.Add(jsonutils.NewString(secGrpObj.GetId()), "secgrp_id")
	}

	/*
		TODO
		group
		for idx := 0; data.Contains(fmt.Sprintf("srvtag.%d", idx)); idx += 1 {

		}*/

	data, err = GetDriver(hypervisor).ValidateCreateData(ctx, userCred, data)

	data, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
	if err != nil {
		return nil, err
	}

	if !jsonutils.QueryBoolean(data, "is_system", false) {
		err = manager.checkCreateQuota(ctx, userCred, ownerProjId, data)
		if err != nil {
			return nil, err
		}
	}

	data.Add(jsonutils.NewString(ownerProjId), "owner_tenant_id")
	return data, nil
}

func (manager *SGuestManager) checkCreateQuota(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, data *jsonutils.JSONDict) error {
	req := getGuestResourceRequirements(ctx, userCred, data, 1)
	err := QuotaManager.CheckSetPendingQuota(ctx, userCred, ownerProjId, &req)
	if err != nil {
		return httperrors.NewOutOfQuotaError(err.Error())
	} else {
		return nil
	}
}

func (self *SGuest) checkUpdateQuota(ctx context.Context, userCred mcclient.TokenCredential, vcpuCount int, vmemSize int) error {
	req := SQuota{}

	if vcpuCount > 0 && vcpuCount > int(self.VcpuCount) {
		req.Cpu = vcpuCount - int(self.VcpuCount)
	}

	if vmemSize > 0 && vmemSize > self.VmemSize {
		req.Memory = vmemSize - self.VmemSize
	}

	_, err := QuotaManager.CheckQuota(ctx, userCred, self.ProjectId, &req)

	return err
}

func getGuestResourceRequirements(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, count int) SQuota {
	vcpuCount, _ := data.Int("vcpu_count")
	if vcpuCount == 0 {
		vcpuCount = 1
	}

	vmemSize, _ := data.Int("vmem_size")

	diskSize := 0

	for idx := 0; data.Contains(fmt.Sprintf("disk.%d", idx)); idx += 1 {
		dataJson, _ := data.Get(fmt.Sprintf("disk.%d", idx))
		diskConfig, _ := parseDiskInfo(ctx, userCred, dataJson)
		diskSize += diskConfig.Size
	}

	devCount := 0
	for idx := 0; data.Contains(fmt.Sprintf("isolated_device.%d", idx)); idx += 1 {
		devCount += 1
	}

	eNicCnt := 0
	iNicCnt := 0
	eBw := 0
	iBw := 0
	for idx := 0; data.Contains(fmt.Sprintf("net.%d", idx)); idx += 1 {
		netJson, _ := data.Get(fmt.Sprintf("net.%d", idx))
		netConfig, _ := parseNetworkInfo(userCred, netJson)
		if isExitNetworkInfo(netConfig) {
			eNicCnt += 1
			eBw += netConfig.BwLimit
		} else {
			iNicCnt += 1
			iBw += netConfig.BwLimit
		}
	}
	return SQuota{
		Cpu:            int(vcpuCount) * count,
		Memory:         int(vmemSize) * count,
		Storage:        diskSize * count,
		Port:           iNicCnt * count,
		Eport:          eNicCnt * count,
		Bw:             iBw * count,
		Ebw:            eBw * count,
		IsolatedDevice: devCount * count,
	}
}

func (guest *SGuest) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	guest.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)

	osProfileJson, _ := data.Get("__os_profile__")
	if osProfileJson != nil {
		guest.setOSProfile(ctx, userCred, osProfileJson)
	}
}

func (manager *SGuestManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	pendingUsage := getGuestResourceRequirements(ctx, userCred, data, len(items))

	taskItems := make([]db.IStandaloneModel, len(items))
	for i, t := range items {
		taskItems[i] = t.(db.IStandaloneModel)
	}
	params := data.(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewParallelTask(ctx, "GuestBatchCreateTask", taskItems, userCred, params, "", "", &pendingUsage)
	if err != nil {
		log.Errorf("GuestBatchCreateTask newTask error %s", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (guest *SGuest) GetGroups() []SGroupguest {
	guestgroups := make([]SGroupguest, 0)
	q := GroupguestManager.Query().Equals("guest_id", guest.Id)
	err := db.FetchModelObjects(GroupguestManager, q, &guestgroups)
	if err != nil {
		log.Errorf("GetGroups fail %s", err)
		return nil
	}
	return guestgroups
}

func (self *SGuest) getBandwidth(isExit bool) int {
	bw := 0
	networks := self.GetNetworks()
	if networks != nil && len(networks) > 0 {
		for i := 0; i < len(networks); i += 1 {
			if networks[i].IsExit() == isExit {
				bw += networks[i].getBandwidth()
			}
		}
	}
	return bw
}

func (self *SGuest) getExtBandwidth() int {
	return self.getBandwidth(true)
}

func (self *SGuest) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)

	if userCred.IsSystemAdmin() {
		host := self.GetHost()
		if host != nil {
			extra.Add(jsonutils.NewString(host.Name), "host")
		}
	}
	extra.Add(jsonutils.NewString(strings.Join(self.getRealIPs(), ",")), "ips")
	extra.Add(jsonutils.NewInt(int64(self.getDiskSize())), "disk")
	// flavor??
	// extra.Add(jsonutils.NewString(self.getFlavorName()), "flavor")
	extra.Add(jsonutils.NewString(self.getKeypairName()), "keypair")
	extra.Add(jsonutils.NewInt(int64(self.getExtBandwidth())), "ext_bw")
	zone := self.getZone()
	if zone != nil {
		extra.Add(jsonutils.NewString(zone.Id), "zone_id")
		extra.Add(jsonutils.NewString(zone.Name), "zone")
	}
	extra.Add(jsonutils.NewString(self.getSecgroupName()), "secgroup")

	if self.PendingDeleted {
		pendingDeletedAt := self.PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
		extra.Add(jsonutils.NewString(timeutils.FullIsoTime(pendingDeletedAt)), "auto_delete_at")
	}

	return extra
}

func (self *SGuest) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	extra.Add(jsonutils.NewString(self.getNetworksDetails()), "networks")
	extra.Add(jsonutils.NewString(self.getDisksDetails()), "disks")
	extra.Add(jsonutils.NewInt(int64(self.getDiskSize())), "disk")
	cdrom := self.getCdrom()
	if cdrom != nil {
		extra.Add(jsonutils.NewString(cdrom.GetDetails()), "cdrom")
	}
	// extra.Add(jsonutils.NewString(self.getFlavorName()), "flavor")
	extra.Add(jsonutils.NewString(self.getKeypairName()), "keypair")
	extra.Add(jsonutils.NewString(self.getSecgroupName()), "secgroup")
	extra.Add(jsonutils.NewString(strings.Join(self.getIPs(), ",")), "ips")
	extra.Add(jsonutils.NewString(self.getSecurityRules()), "security_rules")
	extra.Add(jsonutils.NewString(self.getIsolatedDeviceDetails()), "isolated_devices")
	osName := self.GetOS()
	if len(osName) > 0 {
		extra.Add(jsonutils.NewString(osName), "os_name")
	}
	if userCred.IsSystemAdmin() {
		host := self.GetHost()
		if host != nil {
			extra.Add(jsonutils.NewString(host.GetName()), "host")
		}
		extra.Add(jsonutils.NewString(self.getAdminSecurityRules()), "admin_security_rules")
	}
	zone := self.getZone()
	if zone != nil {
		extra.Add(jsonutils.NewString(zone.GetId()), "zone_id")
		extra.Add(jsonutils.NewString(zone.GetName()), "zone")
	}
	return extra
}

func (self *SGuest) getNetworksDetails() string {
	var buf bytes.Buffer
	for _, nic := range self.GetNetworks() {
		buf.WriteString(nic.GetDetailedString())
		buf.WriteString("\n")
	}
	return buf.String()
}

func (self *SGuest) getDisksDetails() string {
	var buf bytes.Buffer
	for _, disk := range self.GetDisks() {
		buf.WriteString(disk.GetDetailedString())
		buf.WriteString("\n")
	}
	return buf.String()
}

func (self *SGuest) getIsolatedDeviceDetails() string {
	var buf bytes.Buffer
	for _, dev := range self.GetIsolatedDevices() {
		buf.WriteString(dev.getDetailedString())
		buf.WriteString("\n")
	}
	return buf.String()
}

func (self *SGuest) getDiskSize() int {
	size := 0
	for _, disk := range self.GetDisks() {
		size += disk.GetDisk().DiskSize
	}
	return size
}

func (self *SGuest) getCdrom() *SGuestcdrom {
	cdrom := SGuestcdrom{}
	cdrom.SetModelManager(GuestcdromManager)

	err := GuestcdromManager.Query().Equals("id", self.Id).First(&cdrom)
	if err != nil {
		if err == sql.ErrNoRows {
			cdrom.Id = self.Id
			err = GuestcdromManager.TableSpec().Insert(&cdrom)
			if err != nil {
				log.Errorf("insert cdrom fail %s", err)
				return nil
			}
			return &cdrom
		} else {
			log.Errorf("getCdrom query fail %s", err)
			return nil
		}
	} else {
		return &cdrom
	}
}

func (self *SGuest) getKeypair() *SKeypair {
	if len(self.KeypairId) > 0 {
		keypair, _ := KeypairManager.FetchById(self.KeypairId)
		if keypair != nil {
			return keypair.(*SKeypair)
		}
	}
	return nil
}

func (self *SGuest) getKeypairName() string {
	keypair := self.getKeypair()
	if keypair != nil {
		return keypair.Name
	}
	return ""
}

func (self *SGuest) getNotifyIps() []string {
	ips := self.getRealIPs()
	vips := self.getVirtualIPs()
	if vips != nil {
		ips = append(ips, vips...)
	}
	return ips
}

func (self *SGuest) getRealIPs() []string {
	ips := make([]string, 0)
	for _, nic := range self.GetNetworks() {
		if !nic.Virtual {
			ips = append(ips, nic.IpAddr)
		}
	}
	return ips
}

func (self *SGuest) IsExitOnly() bool {
	for _, ip := range self.getRealIPs() {
		addr, _ := netutils.NewIPV4Addr(ip)
		if !netutils.IsExitAddress(addr) {
			return false
		}
	}
	return true
}

func (self *SGuest) getVirtualIPs() []string {
	ips := make([]string, 0)
	for _, guestgroup := range self.GetGroups() {
		group := guestgroup.GetGroup()
		for _, groupnetwork := range group.GetNetworks() {
			ips = append(ips, groupnetwork.IpAddr)
		}
	}
	return ips
}

func (self *SGuest) getIPs() []string {
	ips := self.getRealIPs()
	vips := self.getVirtualIPs()
	ips = append(ips, vips...)
	return ips
}

func (self *SGuest) getZone() *SZone {
	host := self.GetHost()
	if host != nil {
		return host.GetZone()
	}
	return nil
}

func (self *SGuest) GetOS() string {
	if len(self.OsType) > 0 {
		return self.OsType
	}
	return self.GetMetadata("os_name", nil)
}

func (self *SGuest) IsLinux() bool {
	os := self.GetOS()
	if strings.HasPrefix(strings.ToLower(os), "lin") {
		return true
	} else {
		return false
	}
}

func (self *SGuest) IsWindows() bool {
	os := self.GetOS()
	if strings.HasPrefix(strings.ToLower(os), "win") {
		return true
	} else {
		return false
	}
}

func (self *SGuest) getSecgroup() *SSecurityGroup {
	return SecurityGroupManager.FetchSecgroupById(self.SecgrpId)
}

func (self *SGuest) getAdminSecgroup() *SSecurityGroup {
	return SecurityGroupManager.FetchSecgroupById(self.AdminSecgrpId)
}

func (self *SGuest) getSecgroupName() string {
	secgrp := self.getSecgroup()
	if secgrp != nil {
		return secgrp.GetName()
	}
	return ""
}

func (self *SGuest) getAdminSecgroupName() string {
	secgrp := self.getAdminSecgroup()
	if secgrp != nil {
		return secgrp.GetName()
	}
	return ""
}

func (self *SGuest) getSecurityRules() string {
	secgrp := self.getSecgroup()
	if secgrp != nil {
		return secgrp.getSecurityRuleString()
	} else {
		return options.Options.DefaultSecurityRules
	}
}

func (self *SGuest) getAdminSecurityRules() string {
	secgrp := self.getAdminSecgroup()
	if secgrp != nil {
		return secgrp.getSecurityRuleString()
	} else {
		return options.Options.DefaultAdminSecurityRules
	}
}

func (self *SGuest) GetIsolatedDevices() []SIsolatedDevice {
	return IsolatedDeviceManager.findAttachedDevicesOfGuest(self)
}

func (self *SGuest) syncWithCloudVM(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, extVM cloudprovider.ICloudVM) error {
	diff, err := GuestManager.TableSpec().Update(self, func() error {

		self.Name = extVM.GetName()
		self.Status = extVM.GetStatus()
		self.VcpuCount = extVM.GetVcpuCount()
		self.VmemSize = extVM.GetVmemSizeMB()
		self.BootOrder = extVM.GetBootOrder()
		self.Vga = extVM.GetVga()
		self.Vdi = extVM.GetVdi()
		self.OsType = extVM.GetOSType()
		self.Bios = extVM.GetBios()
		self.Machine = extVM.GetMachine()
		self.HostId = host.Id
		self.ProjectId = userCred.GetProjectId()
		self.Hypervisor = extVM.GetHypervisor()

		self.IsEmulated = extVM.IsEmulated()

		return nil
	})
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	if diff != nil {
		diffStr := sqlchemy.UpdateDiffString(diff)
		if len(diffStr) > 0 {
			db.OpsLog.LogEvent(self, db.ACT_UPDATE, diffStr, userCred)
		}
	}
	return nil
}

func (manager *SGuestManager) newCloudVM(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, extVM cloudprovider.ICloudVM) (*SGuest, error) {

	guest := SGuest{}
	guest.SetModelManager(manager)

	guest.Status = extVM.GetStatus()
	guest.ExternalId = extVM.GetGlobalId()
	guest.Name = extVM.GetName()
	guest.VcpuCount = extVM.GetVcpuCount()
	guest.VmemSize = extVM.GetVmemSizeMB()
	guest.BootOrder = extVM.GetBootOrder()
	guest.Vga = extVM.GetVga()
	guest.Vdi = extVM.GetVdi()
	guest.OsType = extVM.GetOSType()
	guest.Bios = extVM.GetBios()
	guest.Machine = extVM.GetMachine()
	guest.Hypervisor = extVM.GetHypervisor()

	guest.IsEmulated = extVM.IsEmulated()

	guest.HostId = host.Id
	guest.ProjectId = userCred.GetProjectId()

	err := manager.TableSpec().Insert(&guest)
	if err != nil {
		log.Errorf("Insert fail %s", err)
	}
	return &guest, nil
}

func (manager *SGuestManager) TotalCount(
	projectId string, rangeObj db.IStandaloneModel,
	status []string, hypervisor string,
	includeSystem bool, pendingDelete bool, hostType string,
) SGuestCountStat {
	return totalGuestResourceCount(projectId, rangeObj, status, hypervisor, includeSystem, pendingDelete, hostType)
}

func (self *SGuest) detachNetwork(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork, reserve bool, deploy bool) error {
	// Portmaps.delete_guest_network_portmaps(self, user_cred,
	//                                                    network_id=net.id)
	err := GuestnetworkManager.DeleteGuestNics(ctx, self, userCred, network, reserve)
	if err != nil {
		return err
	}
	host := self.GetHost()
	if host != nil {
		host.ClearSchedDescCache() // ignore error
	}
	if deploy {
		self.StartGuestDeployTask(ctx, userCred, nil, "deploy", "")
	}
	return nil
}

func (self *SGuest) isAttach2Network(net *SNetwork) bool {
	q := GuestnetworkManager.Query()
	q = q.Equals("guest_id", self.Id).Equals("network_id", net.Id)
	return q.Count() > 0
}

func (self *SGuest) getMaxNicIndex() int8 {
	nics := self.GetNetworks()
	return int8(len(nics))
}

func (self *SGuest) setOSProfile(ctx context.Context, userCred mcclient.TokenCredential, profile jsonutils.JSONObject) error {
	return self.SetMetadata(ctx, "__os_profile__", profile, userCred)
}

func (self *SGuest) getOSProfile() osprofile.SOSProfile {
	osName := self.GetOS()
	osProf := osprofile.GetOSProfile(osName, self.Hypervisor)
	val := self.GetMetadata("__os_profile__", nil)
	if len(val) > 0 {
		jsonVal, _ := jsonutils.ParseString(val)
		if jsonVal != nil {
			jsonVal.Unmarshal(&osProf)
		}
	}
	return osProf
}

func (self *SGuest) Attach2Network(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork, pendingUsage quotas.IQuota,
	address string, mac string, driver string, bwLimit int, virtual bool, index int8, reserved bool, allocDir IPAddlocationDirection, requireDesignatedIP bool) error {
	if self.isAttach2Network(network) {
		return fmt.Errorf("Guest has been attached to network %s", network.Name)
	}
	if index < 0 {
		index = self.getMaxNicIndex()
	}
	if len(driver) == 0 {
		osProf := self.getOSProfile()
		driver = osProf.NetDriver
	}
	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)

	guestnic, err := GuestnetworkManager.newGuestNetwork(ctx, userCred, self, network,
		index, address, mac, driver, bwLimit, virtual, reserved,
		allocDir, requireDesignatedIP)
	if err != nil {
		return err
	}
	network.updateDnsRecord(guestnic, true)
	network.updateGuestNetmap(guestnic)
	bwLimit = guestnic.getBandwidth()
	if pendingUsage != nil {
		cancelUsage := SQuota{}
		if network.IsExitNetwork() {
			cancelUsage.Eport = 1
			cancelUsage.Ebw = bwLimit
		} else {
			cancelUsage.Port = 1
			cancelUsage.Bw = bwLimit
		}
		err = QuotaManager.CancelPendingUsage(ctx, userCred, self.ProjectId, pendingUsage, &cancelUsage)
		if err != nil {
			return err
		}
	}
	notes := jsonutils.NewDict()
	notes.Add(jsonutils.NewString(address), "ip_addr")
	db.OpsLog.LogAttachEvent(self, network, userCred, notes)
	return nil
}

type sRemoveGuestnic struct {
	nic     *SGuestnetwork
	reserve bool
}

type sAddGuestnic struct {
	nic     cloudprovider.ICloudNic
	net     *SNetwork
	reserve bool
}

func getCloudNicNetwork(vnic cloudprovider.ICloudNic, host *SHost) (*SNetwork, error) {
	vnet := vnic.GetINetwork()
	if vnet == nil {
		ip := vnic.GetIP()
		if len(ip) == 0 {
			return nil, fmt.Errorf("Cannot find inetwork for vnics %s %s", vnic.GetMAC(), vnic.GetIP())
		} else {
			// find network by IP
			return host.getNetworkOfIPOnHost(vnic.GetIP())
		}
	}
	localNetObj, err := NetworkManager.FetchByExternalId(vnet.GetGlobalId())
	if err != nil {
		return nil, fmt.Errorf("Cannot find network of external_id %s", vnet.GetGlobalId())
	}
	localNet := localNetObj.(*SNetwork)
	return localNet, nil
}

func (self *SGuest) SyncVMNics(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, vnics []cloudprovider.ICloudNic) compare.SyncResult {
	result := compare.SyncResult{}

	guestnics := self.GetNetworks()
	removed := make([]sRemoveGuestnic, 0)
	adds := make([]sAddGuestnic, 0)

	for i := 0; i < len(guestnics) || i < len(vnics); i += 1 {
		if i < len(guestnics) && i < len(vnics) {
			localNet, err := getCloudNicNetwork(vnics[i], host)
			if err != nil {
				log.Errorf("%s", err)
				result.Error(err)
				return result
			}
			if guestnics[i].NetworkId == localNet.Id {
				if guestnics[i].MacAddr == vnics[i].GetMAC() {
					if guestnics[i].IpAddr == vnics[i].GetIP() { // nothing changes
						// do nothing
					} else if len(vnics[i].GetIP()) > 0 {
						// ip changed
						removed = append(removed, sRemoveGuestnic{nic: &guestnics[i]})
						adds = append(adds, sAddGuestnic{nic: vnics[i], net: localNet})
					} else {
						// do nothing
						// vm maybe turned off, ignore the case
					}
				} else {
					reserve := false
					if len(guestnics[i].IpAddr) > 0 && guestnics[i].IpAddr == vnics[i].GetIP() {
						// mac changed
						reserve = true
					}
					removed = append(removed, sRemoveGuestnic{nic: &guestnics[i], reserve: reserve})
					adds = append(adds, sAddGuestnic{nic: vnics[i], net: localNet, reserve: reserve})
				}
			} else {
				removed = append(removed, sRemoveGuestnic{nic: &guestnics[i]})
				adds = append(adds, sAddGuestnic{nic: vnics[i], net: localNet})
			}
		} else if i < len(guestnics) {
			removed = append(removed, sRemoveGuestnic{nic: &guestnics[i]})
		} else if i < len(vnics) {
			localNet, err := getCloudNicNetwork(vnics[i], host)
			if err != nil {
				log.Errorf("%s", err) // ignore this case
			} else {
				adds = append(adds, sAddGuestnic{nic: vnics[i], net: localNet})
			}
		}
	}

	for _, remove := range removed {
		err := self.detachNetwork(ctx, userCred, remove.nic.GetNetwork(), remove.reserve, false)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for _, add := range adds {
		if len(add.nic.GetIP()) == 0 {
			continue // cannot determine which network it attached to
		}
		if add.net == nil {
			continue // cannot determine which network it attached to
		}
		err := self.Attach2Network(ctx, userCred, add.net, nil, add.nic.GetIP(),
			add.nic.GetMAC(), add.nic.GetDriver(), 0, false, -1, add.reserve, IPAllocationDefault, true)
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}

	return result
}

func (self *SGuest) isAttach2Disk(disk *SDisk) bool {
	q := GuestdiskManager.Query().Equals("disk_id", disk.Id).Equals("guest_id", self.Id)
	return q.Count() > 0
}

func (self *SGuest) getMaxDiskIndex() int8 {
	guestdisks := self.GetDisks()
	return int8(len(guestdisks))
}

func (self *SGuest) attach2Disk(disk *SDisk, userCred mcclient.TokenCredential, driver string, cache string, mountpoint string) error {
	if self.isAttach2Disk(disk) {
		return fmt.Errorf("Guest has been attached to disk")
	}
	index := self.getMaxDiskIndex()
	if len(driver) == 0 {
		osProf := self.getOSProfile()
		driver = osProf.DiskDriver
	}
	guestdisk := SGuestdisk{}
	guestdisk.SetModelManager(GuestdiskManager)

	guestdisk.DiskId = disk.Id
	guestdisk.GuestId = self.Id
	guestdisk.Index = index
	err := guestdisk.DoSave(driver, cache, mountpoint)
	if err == nil {
		db.OpsLog.LogAttachEvent(self, disk, userCred, nil)
	}
	return err
}

type sSyncDiskPair struct {
	disk  *SDisk
	vdisk cloudprovider.ICloudDisk
}

func (self *SGuest) SyncVMDisks(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, vdisks []cloudprovider.ICloudDisk) compare.SyncResult {
	result := compare.SyncResult{}

	newdisks := make([]sSyncDiskPair, 0)
	for i := 0; i < len(vdisks); i += 1 {
		if len(vdisks[i].GetGlobalId()) == 0 {
			continue
		}
		disk, err := DiskManager.syncCloudDisk(userCred, vdisks[i])
		if err != nil {
			result.Error(err)
			return result
		}
		newdisks = append(newdisks, sSyncDiskPair{disk: disk, vdisk: vdisks[i]})
	}

	needRemoves := make([]SGuestdisk, 0)

	guestdisks := self.GetDisks()
	for i := 0; i < len(guestdisks); i += 1 {
		find := false
		for j := 0; j < len(newdisks); j += 1 {
			if newdisks[j].disk.Id == guestdisks[i].DiskId {
				find = true
				break
			}
		}
		if !find {
			needRemoves = append(needRemoves, guestdisks[i])
		}
	}

	needAdds := make([]sSyncDiskPair, 0)

	for i := 0; i < len(newdisks); i += 1 {
		find := false
		for j := 0; j < len(guestdisks); j += 1 {
			if newdisks[i].disk.Id == guestdisks[j].DiskId {
				find = true
				break
			}
		}
		if !find {
			needAdds = append(needAdds, newdisks[i])
		}
	}

	for i := 0; i < len(needRemoves); i += 1 {
		err := needRemoves[i].Detach(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}
	for i := 0; i < len(needAdds); i += 1 {
		vdisk := needAdds[i].vdisk
		err := self.attach2Disk(needAdds[i].disk, userCred, vdisk.GetDriver(), vdisk.GetCacheMode(), vdisk.GetMountpoint())
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func filterGuestByRange(q *sqlchemy.SQuery, rangeObj db.IStandaloneModel, hostType string) *sqlchemy.SQuery {
	hosts := HostManager.Query().SubQuery()
	q = q.Join(hosts, sqlchemy.AND(
		sqlchemy.Equals(hosts.Field("id"), q.Field("host_id")),
		sqlchemy.IsFalse(hosts.Field("deleted")),
		sqlchemy.IsTrue(hosts.Field("enabled")),
		sqlchemy.Equals(hosts.Field("host_status"), HOST_ONLINE)))
	hostTypes := []string{}
	if len(hostType) != 0 {
		hostTypes = append(hostTypes, hostType)
	}
	q = AttachUsageQuery(q, hosts, hosts.Field("id"), hostTypes, rangeObj)
	return q
}

type SGuestCountStat struct {
	TotalGuestCount    int
	TotalCpuCount      int
	TotalMemSize       int
	TotalDiskSize      int
	TotalIsolatedCount int
}

func totalGuestResourceCount(projectId string, rangeObj db.IStandaloneModel, status []string, hypervisor string,
	includeSystem bool, pendingDelete bool, hostType string) SGuestCountStat {

	guestdisks := GuestdiskManager.Query().SubQuery()
	disks := DiskManager.Query().SubQuery()

	diskQuery := guestdisks.Query(guestdisks.Field("guest_id"), sqlchemy.SUM("guest_disk_size", disks.Field("disk_size")))
	diskQuery = diskQuery.Join(disks, sqlchemy.AND(sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id")),
		sqlchemy.IsFalse(disks.Field("deleted"))))
	diskQuery = diskQuery.GroupBy(guestdisks.Field("guest_id"))

	diskSubQuery := diskQuery.SubQuery()

	isolated := IsolatedDeviceManager.Query().SubQuery()

	isoDevQuery := isolated.Query(isolated.Field("guest_id"), sqlchemy.COUNT("device_sum"))
	isoDevQuery = isoDevQuery.Filter(sqlchemy.IsNotNull(isolated.Field("guest_id")))
	isoDevQuery = isoDevQuery.GroupBy(isolated.Field("guest_id"))

	isoDevSubQuery := isoDevQuery.SubQuery()

	guests := GuestManager.Query().SubQuery()

	q := guests.Query(sqlchemy.COUNT("total_guest_count"),
		sqlchemy.SUM("total_cpu_count", guests.Field("vcpu_count")),
		sqlchemy.SUM("total_mem_size", guests.Field("vmem_size")),
		sqlchemy.SUM("total_disk_size", diskSubQuery.Field("guest_disk_size")),
		sqlchemy.SUM("total_isolated_count", isoDevSubQuery.Field("device_sum")))

	q = q.LeftJoin(diskSubQuery, sqlchemy.Equals(diskSubQuery.Field("guest_id"), guests.Field("id")))

	q = q.LeftJoin(isoDevSubQuery, sqlchemy.Equals(isoDevSubQuery.Field("guest_id"), guests.Field("id")))

	q = filterGuestByRange(q, rangeObj, hostType)

	if len(projectId) > 0 {
		q = q.Filter(sqlchemy.Equals(guests.Field("tenant_id"), projectId))
	}
	if len(status) > 0 {
		q = q.Filter(sqlchemy.In(guests.Field("status"), status))
	}
	if len(hypervisor) > 0 {
		q = q.Filter(sqlchemy.Equals(guests.Field("hypervisor"), hypervisor))
	}
	if !includeSystem {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(guests.Field("is_system")), sqlchemy.IsFalse(guests.Field("is_system"))))
	}
	if pendingDelete {
		q = q.Filter(sqlchemy.IsTrue(guests.Field("pending_deleted")))
	} else {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(guests.Field("pending_deleted")), sqlchemy.IsFalse(guests.Field("pending_deleted"))))
	}
	stat := SGuestCountStat{}
	row := q.Row()
	err := q.Row2Struct(row, &stat)
	if err != nil {
		log.Errorf("%s", err)
	}
	return stat
}

func (self *SGuest) getDefaultNetworkConfig() *SNetworkConfig {
	netConf := SNetworkConfig{}
	netConf.BwLimit = options.Options.DefaultBandwidth
	osProf := self.getOSProfile()
	netConf.Driver = osProf.NetDriver
	return &netConf
}

func (self *SGuest) CreateNetworksOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, data *jsonutils.JSONDict, pendingUsage quotas.IQuota) error {
	idx := 0
	for idx = 0; data.Contains(fmt.Sprintf("net.%d", idx)); idx += 1 {
		netJson, err := data.Get(fmt.Sprintf("net.%d", idx))
		if err != nil {
			return err
		}
		netConfig, err := parseNetworkInfo(userCred, netJson)
		if err != nil {
			return err
		}
		err = self.attach2NetworkDesc(ctx, userCred, host, netConfig, pendingUsage)
		if err != nil {
			return err
		}
	}
	if idx == 0 {
		netConfig := self.getDefaultNetworkConfig()
		return self.attach2RandomNetwork(ctx, userCred, host, netConfig, pendingUsage)
	}
	return nil
}

func (self *SGuest) attach2NetworkDesc(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *SNetworkConfig, pendingUsage quotas.IQuota) error {
	var err1, err2 error
	if len(netConfig.Network) > 0 {
		err1 = self.attach2NamedNetworkDesc(ctx, userCred, host, netConfig, pendingUsage)
		if err1 == nil {
			return nil
		}
	}
	err2 = self.attach2RandomNetwork(ctx, userCred, host, netConfig, pendingUsage)
	if err2 == nil {
		return nil
	}
	if err1 != nil {
		return fmt.Errorf("%s/%s", err1, err2)
	} else {
		return err2
	}
}

func (self *SGuest) attach2NamedNetworkDesc(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *SNetworkConfig, pendingUsage quotas.IQuota) error {
	driver := self.GetDriver()
	net, mac, idx, allocDir := driver.GetNamedNetworkConfiguration(self, userCred, host, netConfig)
	if net != nil {
		err := self.Attach2Network(ctx, userCred, net, pendingUsage, netConfig.Address, mac, netConfig.Driver, netConfig.BwLimit, netConfig.Vip, idx, netConfig.Reserved, allocDir, false)
		if err != nil {
			return err
		} else {
			return nil
		}
	} else {
		return fmt.Errorf("Network %s not available", netConfig.Network)
	}
}

func (self *SGuest) attach2RandomNetwork(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *SNetworkConfig, pendingUsage quotas.IQuota) error {
	driver := self.GetDriver()
	return driver.Attach2RandomNetwork(self, ctx, userCred, host, netConfig, pendingUsage)
}

func (self *SGuest) CreateDisksOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, data *jsonutils.JSONDict, pendingUsage quotas.IQuota) error {
	for idx := 0; data.Contains(fmt.Sprintf("disk.%d", idx)); idx += 1 {
		diskJson, err := data.Get(fmt.Sprintf("disk.%d", idx))
		if err != nil {
			return err
		}
		diskConfig, err := parseDiskInfo(ctx, userCred, diskJson)
		if err != nil {
			return err
		}
		disk, err := self.createDiskOnHost(ctx, userCred, host, diskConfig, pendingUsage)
		if err != nil {
			return err
		}
		data.Add(jsonutils.NewString(disk.Id), fmt.Sprintf("disk.%d.id", idx))
	}
	return nil
}

func (self *SGuest) createDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage, diskConfig *SDiskConfig, pendingUsage quotas.IQuota) (*SDisk, error) {
	lockman.LockObject(ctx, storage)
	defer lockman.LockObject(ctx, storage)

	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)

	diskName := fmt.Sprintf("vdisk_%s_%d", self.Name, time.Now().UnixNano())
	disk, err := storage.createDisk(diskName, diskConfig, userCred, self.ProjectId, true, self.IsSystem)

	if err != nil {
		return nil, err
	}

	cancelUsage := SQuota{}
	cancelUsage.Storage = disk.DiskSize
	err = QuotaManager.CancelPendingUsage(ctx, userCred, self.ProjectId, pendingUsage, &cancelUsage)

	return disk, nil
}

func (self *SGuest) createDiskOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, diskConfig *SDiskConfig, pendingUsage quotas.IQuota) (*SDisk, error) {
	storage := self.GetDriver().ChooseHostStorage(host, diskConfig.Backend)
	if storage == nil {
		return nil, fmt.Errorf("No storage to create disk")
	}

	disk, err := self.createDiskOnStorage(ctx, userCred, storage, diskConfig, pendingUsage)
	if err != nil {
		return nil, err
	}

	err = self.attach2Disk(disk, userCred, diskConfig.Driver, diskConfig.Cache, diskConfig.Mountpoint)
	return disk, err
}

func (self *SGuest) CreateIsolatedDeviceOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, data *jsonutils.JSONDict, pendingUsage quotas.IQuota) error {
	for idx := 0; data.Contains(fmt.Sprintf("isolated_device.%d", idx)); idx += 1 {
		devJson, err := data.Get(fmt.Sprintf("isolated_device.%d", idx))
		if err != nil {
			return err
		}
		devConfig, err := IsolatedDeviceManager.parseDeviceInfo(userCred, devJson)
		if err != nil {
			return err
		}
		err = self.createIsolatedDeviceOnHost(ctx, userCred, host, devConfig, pendingUsage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) createIsolatedDeviceOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, devConfig *SIsolatedDeviceConfig, pendingUsage quotas.IQuota) error {
	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)

	err := IsolatedDeviceManager.attachHostDeviceToGuestByDesc(self, host, devConfig, userCred)
	if err != nil {
		return err
	}

	cancelUsage := SQuota{IsolatedDevice: 1}
	err = QuotaManager.CancelPendingUsage(ctx, userCred, self.ProjectId, pendingUsage, &cancelUsage)
	return err
}

func (self *SGuest) attachIsolatedDevice(userCred mcclient.TokenCredential, dev *SIsolatedDevice) error {
	if len(dev.GuestId) > 0 {
		return fmt.Errorf("Isolated device already attached to another guest: %s", dev.GuestId)
	}
	if dev.HostId != self.HostId {
		return fmt.Errorf("Isolated device and guest are not located in the same host")
	}
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		dev.GuestId = self.Id
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_GUEST_ATTACH_ISOLATED_DEVICE, dev.GetShortDesc(), userCred)
	return nil
}

func (self *SGuest) JoinGroups(userCred mcclient.TokenCredential, params *jsonutils.JSONDict) {
	// TODO
}

func (self *SGuest) StartGuestCreateTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, pendingUsage quotas.IQuota, parentTaskId string) error {
	return self.GetDriver().StartGuestCreateTask(self, ctx, userCred, params, pendingUsage, parentTaskId)
}

type SGuestDiskCategory struct {
	Root *SDisk
	Swap []*SDisk
	Data []*SDisk
}

func (self *SGuest) CategorizeDisks() SGuestDiskCategory {
	diskCat := SGuestDiskCategory{}
	guestdisks := self.GetDisks()
	if guestdisks == nil {
		log.Errorf("no disk for this server!!!")
		return diskCat
	}
	for _, gd := range guestdisks {
		if diskCat.Root == nil {
			diskCat.Root = gd.GetDisk()
		} else {
			disk := gd.GetDisk()
			if disk.FsFormat == "swap" {
				diskCat.Swap = append(diskCat.Swap, disk)
			} else {
				diskCat.Data = append(diskCat.Data, disk)
			}
		}
	}
	return diskCat
}

type SGuestNicCategory struct {
	InternalNics []SGuestnetwork
	ExternalNics []SGuestnetwork
}

func (self *SGuest) CategorizeNics() SGuestNicCategory {
	netCat := SGuestNicCategory{}

	guestnics := self.GetNetworks()
	if guestnics == nil {
		log.Errorf("no nics for this server!!!")
		return netCat
	}

	for _, gn := range guestnics {
		if gn.IsExit() {
			netCat.ExternalNics = append(netCat.ExternalNics, gn)
		} else {
			netCat.InternalNics = append(netCat.InternalNics, gn)
		}
	}
	return netCat
}

func (self *SGuest) StartGuestDeployTask(ctx context.Context, userCred mcclient.TokenCredential, kwargs *jsonutils.JSONDict, action string, parentTaskId string) error {
	self.SetStatus(userCred, VM_START_DEPLOY, "")
	kwargs.Add(jsonutils.NewString(action), "deploy_action")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestDeployTask", self, userCred, kwargs, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) NotifyServerEvent(event string, priority string, loginInfo bool) error {
	meta, err := self.GetAllMetadata(nil)
	if err != nil {
		return err
	}
	kwargs := jsonutils.NewDict()
	kwargs.Add(jsonutils.NewString(self.Name), "name")
	if loginInfo {
		kwargs.Add(jsonutils.NewStringArray(self.getNotifyIps()), "ips")
		osName := meta["os_name"]
		if osName == "Windows" {
			kwargs.Add(jsonutils.JSONTrue, "windows")
		}
		loginAccount := meta["login_account"]
		if len(loginAccount) > 0 {
			kwargs.Add(jsonutils.NewString(loginAccount), "account")
		}
		keypair := self.getKeypairName()
		if len(keypair) > 0 {
			kwargs.Add(jsonutils.NewString(keypair), "keypair")
		} else {
			loginKey := meta["login_key"]
			if len(loginKey) > 0 {
				passwd, err := utils.DescryptAESBase64(self.Id, loginKey)
				if err == nil {
					kwargs.Add(jsonutils.NewString(passwd), "password")
				}
			}
		}
	}
	return notifyclient.Notify(self.ProjectId, event, priority, kwargs)
}

func (self *SGuest) NotifyAdminServerEvent(ctx context.Context, event string, priority string) error {
	kwargs := jsonutils.NewDict()
	kwargs.Add(jsonutils.NewString(self.Name), "name")
	tc, _ := self.GetTenantCache(ctx)
	if tc != nil {
		kwargs.Add(jsonutils.NewString(tc.Name), "tenant")
	} else {
		kwargs.Add(jsonutils.NewString(self.ProjectId), "tenant")
	}
	return notifyclient.Notify(options.Options.NotifyAdminUser, event, priority, kwargs)
}

func (self *SGuest) StartGuestStopTask(ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error {
	if len(parentTaskId) == 0 {
		self.SetStatus(userCred, VM_START_STOP, "")
	}
	params := jsonutils.NewDict()
	if isForce {
		params.Add(jsonutils.JSONTrue, "is_force")
	}
	if len(parentTaskId) > 0 {
		params.Add(jsonutils.JSONTrue, "subtask")
	}
	return self.GetDriver().StartGuestStopTask(self, ctx, userCred, params, parentTaskId)
}

func (self *SGuest) insertIso(imageId string) bool {
	cdrom := self.getCdrom()
	return cdrom.insertIso(imageId)
}

func (self *SGuest) insertIsoSucc(imageId string, path string, size int, name string) bool {
	cdrom := self.getCdrom()
	return cdrom.insertIsoSucc(imageId, path, size, name)
}

func (self *SGuest) StartInsertIsoTask(ctx context.Context, imageId string, hostId string, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.insertIso(imageId)

	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(imageId), "image_id")
	data.Add(jsonutils.NewString(hostId), "host_id")

	task, err := taskman.TaskManager.NewTask(ctx, "GuestInsertISOTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartGueststartTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, VM_START_START, "")
	task, err := taskman.TaskManager.NewTask(ctx, "GuestStartTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return self.GetDriver().StartGuestSyncstatusTask(self, ctx, userCred, parentTaskId)
}

func (self *SGuest) StartAutoDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	db.OpsLog.LogEvent(self, db.ACT_DELETE, "auto-delete after stop", userCred)
	return self.StartDeleteGuestTask(ctx, userCred, parentTaskId, false, false)
}

func (self *SGuest) StartDeleteGuestTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, isPurge bool, overridePendingDelete bool) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(self.Status), "guest_status")
	if isPurge {
		params.Add(jsonutils.JSONTrue, "purge")
	}
	if overridePendingDelete {
		params.Add(jsonutils.JSONTrue, "override_pending_delete")
	}
	self.SetStatus(userCred, VM_START_DELETE, "")
	return self.GetDriver().StartDeleteGuestTask(self, ctx, userCred, params, parentTaskId)
}

func (self *SGuest) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsAdmin(userCred)
}

func (self *SGuest) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	host := self.GetHost()
	if host != nil && host.Enabled {
		return nil, httperrors.NewInvalidStatusError("Cannot purge server on enabled host")
	}
	err = self.StartDeleteGuestTask(ctx, userCred, "", true, false)
	return nil, err
}

func (self *SGuest) detachDisk(ctx context.Context, disk *SDisk, userCred mcclient.TokenCredential) {
	guestdisk := self.GetGuestDisk(disk.Id)
	if guestdisk != nil {
		guestdisk.Detach(ctx, userCred)
	}
}

func (self *SGuest) DoPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	for _, guestdisk := range self.GetDisks() {
		disk := guestdisk.GetDisk()
		storage := disk.GetStorage()
		if utils.IsInStringArray(storage.StorageType, sysutils.LOCAL_STORAGE_TYPES) || disk.DiskType == DISK_TYPE_SYS || disk.DiskType == DISK_TYPE_SWAP || self.Hypervisor == HYPERVISOR_ALIYUN {
			disk.DoPendingDelete(ctx, userCred)
		} else {
			self.detachDisk(ctx, disk, userCred)
		}
	}
	self.SVirtualResourceBase.DoPendingDelete(ctx, userCred)
}

func (model *SGuest) AllowPerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SGuest) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.PendingDeleted {
		err := self.DoCancelPendingDelete(ctx, userCred)
		return nil, err
	}
	return nil, nil
}

func (self *SGuest) DoCancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	for _, guestdisk := range self.GetDisks() {
		disk := guestdisk.GetDisk()
		disk.DoCancelPendingDelete(ctx, userCred)
	}
	return self.SVirtualResourceBase.DoCancelPendingDelete(ctx, userCred)
}

func (self *SGuest) StartUndeployGuestTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, targetHostId string) error {
	data := jsonutils.NewDict()
	if len(targetHostId) > 0 {
		data.Add(jsonutils.NewString(targetHostId), "target_host_id")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "GuestUndeployTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SGuest) LeaveAllGroups(userCred mcclient.TokenCredential) {
	// TODO
}

func (self *SGuest) DetachAllNetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	// from clouds.models.portmaps import Portmaps
	// Portmaps.delete_guest_network_portmaps(self, user_cred)
	return GuestnetworkManager.DeleteGuestNics(ctx, self, userCred, nil, false)
}

func (self *SGuest) EjectIso(userCred mcclient.TokenCredential) bool {
	cdrom := self.getCdrom()
	if len(cdrom.ImageId) > 0 {
		imageId := cdrom.ImageId
		if cdrom.ejectIso() {
			db.OpsLog.LogEvent(self, db.ACT_ISO_DETACH, imageId, userCred)
			return true
		}
	}
	return false
}

func (self *SGuest) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// self.SVirtualResourceBase.Delete(ctx, userCred)
	// override
	log.Infof("guest delete do nothing")
	return nil
}

func (self *SGuest) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SGuest) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	overridePendingDelete := false
	purge := false
	if data != nil {
		overridePendingDelete = jsonutils.QueryBoolean(data, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(data, "purge", false)
	}
	if (overridePendingDelete || purge) && !userCred.IsSystemAdmin() {
		return false
	}
	return self.IsOwner(userCred)
}

func (self *SGuest) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	overridePendingDelete := false
	purge := false
	if data != nil {
		overridePendingDelete = jsonutils.QueryBoolean(data, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(data, "purge", false)
	}
	return self.StartDeleteGuestTask(ctx, userCred, "", purge, overridePendingDelete)
}

func (self *SGuest) DeleteAllDisksInDB(ctx context.Context, userCred mcclient.TokenCredential) error {
	for _, guestdisk := range self.GetDisks() {
		disk := guestdisk.GetDisk()
		err := guestdisk.Detach(ctx, userCred)
		if err != nil {
			return err
		}
		db.OpsLog.LogEvent(disk, db.ACT_DELETE, nil, userCred)
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE, nil, userCred)
		err = disk.RealDelete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SGuest) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	self.SetStatus(userCred, VM_SYNCING_STATUS, "perform_syncstatus")
	err := self.StartSyncstatus(ctx, userCred, "")
	return nil, err
}

type SDeployConfig struct {
	Path    string
	Action  string
	Content string
}

func (self *SGuest) GetDeployConfigOnHost(ctx context.Context, host *SHost, params *jsonutils.JSONDict) *jsonutils.JSONDict {
	config := jsonutils.NewDict()

	desc := self.GetDriver().GetJsonDescAtHost(ctx, self, host)
	config.Add(desc, "desc")

	deploys := make([]SDeployConfig, 0)
	for idx := 0; params.Contains(fmt.Sprintf("deploy.%d.path", idx)); idx += 1 {
		path, _ := params.GetString(fmt.Sprintf("deploy.%d.path", idx))
		action, _ := params.GetString(fmt.Sprintf("deploy.%d.action", idx))
		content, _ := params.GetString(fmt.Sprintf("deploy.%d.content", idx))
		deploys = append(deploys, SDeployConfig{Path: path, Action: action, Content: content})
	}

	if len(deploys) > 0 {
		config.Add(jsonutils.Marshal(deploys), "deploys")
	}

	deployAction, _ := params.GetString("deploy_action")
	if len(deployAction) == 0 {
		deployAction = "deploy"
	}

	resetPasswd := true
	if deployAction == "deploy" {
		resetPasswd = jsonutils.QueryBoolean(params, "reset_password", false)
	}

	if resetPasswd {
		config.Add(jsonutils.JSONTrue, "reset_password")
		keypair := self.getKeypair()
		if keypair != nil {
			config.Add(jsonutils.NewString(keypair.PublicKey), "public_key")
		}
	}

	config.Add(jsonutils.NewString(deployAction), "action")

	onFinish := "shutdown"
	if jsonutils.QueryBoolean(params, "auto_start", false) || jsonutils.QueryBoolean(params, "restart", false) {
		onFinish = "none"
	} else if utils.IsInStringArray(self.Status, []string{VM_ADMIN}) {
		onFinish = "none"
	}

	config.Add(jsonutils.NewString(onFinish), "on_finish")

	return config
}

func (self *SGuest) getVga() string {
	if utils.IsInStringArray(self.Vga, []string{"cirrus", "vmware", "qxl"}) {
		return self.Vga
	}
	return "std"
}

func (self *SGuest) GetVdi() string {
	if utils.IsInStringArray(self.Vdi, []string{"vnc", "spice"}) {
		return self.Vdi
	}
	return "vnc"
}

func (self *SGuest) getMachine() string {
	if utils.IsInStringArray(self.Machine, []string{"pc", "q35"}) {
		return self.Machine
	}
	return "pc"
}

func (self *SGuest) getBios() string {
	if utils.IsInStringArray(self.Bios, []string{"BIOS", "UEFI"}) {
		return self.Bios
	}
	return "BIOS"
}

func (self *SGuest) getKvmOptions() string {
	return self.GetMetadata("kvm", nil)
}

func (self *SGuest) getExtraOptions() jsonutils.JSONObject {
	return self.GetMetadataJson("extra_options", nil)
}

func (self *SGuest) GetJsonDescAtHypervisor(ctx context.Context, host *SHost) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()

	desc.Add(jsonutils.NewString(self.Name), "name")
	if len(self.Description) > 0 {
		desc.Add(jsonutils.NewString(self.Description), "description")
	}
	desc.Add(jsonutils.NewString(self.Id), "uuid")
	desc.Add(jsonutils.NewInt(int64(self.VmemSize)), "mem")
	desc.Add(jsonutils.NewInt(int64(self.VcpuCount)), "cpu")
	desc.Add(jsonutils.NewString(self.getVga()), "vga")
	desc.Add(jsonutils.NewString(self.GetVdi()), "vdi")
	desc.Add(jsonutils.NewString(self.getMachine()), "machine")
	desc.Add(jsonutils.NewString(self.getBios()), "bios")
	desc.Add(jsonutils.NewString(self.BootOrder), "boot_order")

	isolatedDevs := IsolatedDeviceManager.generateJsonDescForGuest(self)
	desc.Add(jsonutils.NewArray(isolatedDevs...), "solated_devices")

	jsonNics := make([]jsonutils.JSONObject, 0)
	nics := self.GetNetworks()
	domain := options.Options.DNSDomain

	if nics != nil && len(nics) > 0 {
		for _, nic := range nics {
			nicDesc := nic.getJsonDescAtHost(host)
			jsonNics = append(jsonNics, nicDesc)
			nicDomain, _ := nicDesc.GetString("domain")
			if len(nicDomain) > 0 && len(domain) == 0 {
				domain = nicDomain
			}
		}
	}
	desc.Add(jsonutils.NewArray(jsonNics...), "nics")
	desc.Add(jsonutils.NewString(domain), "domain")

	jsonDisks := make([]jsonutils.JSONObject, 0)
	disks := self.GetDisks()
	if disks != nil && len(disks) > 0 {
		for _, disk := range disks {
			diskDesc := disk.GetJsonDescAtHost(host)
			jsonDisks = append(jsonDisks, diskDesc)
		}
	}
	desc.Add(jsonutils.NewArray(jsonDisks...), "disks")

	cdDesc := self.getCdrom().getJsonDesc()
	if cdDesc != nil {
		desc.Add(cdDesc, "cdrom")
	}

	tc, _ := self.GetTenantCache(ctx)
	if tc != nil {
		desc.Add(jsonutils.NewString(tc.GetName()), "tenant")
	}

	desc.Add(jsonutils.NewString(self.ProjectId), "tenant_id")
	// flavor
	// desc.Add(jsonuitls.NewString(self.getFlavorName()), "flavor")

	keypair := self.getKeypair()
	if keypair != nil {
		desc.Add(jsonutils.NewString(keypair.Name), "keypair")
		desc.Add(jsonutils.NewString(keypair.PublicKey), "pubkey")
	}

	netRoles := self.getNetworkRoles()
	if netRoles != nil && len(netRoles) > 0 {
		desc.Add(jsonutils.NewStringArray(netRoles), "network_roles")
	}

	secGrp := self.getSecgroup()
	if secGrp != nil {
		desc.Add(jsonutils.NewString(secGrp.Name), "secgroup")
	}

	/*
		TODO
		srs := self.getSecurityRuleSet()
		if srs.estimatedSinglePortRuleCount() <= options.FirewallFlowCountLimit {
			rules := self.getSecurityRules()
			if len(rules) > 0 {
				desc.Add(jsonutils.NewString(rules), "security_rules")
			}
			rules = self.getAdminSecurityRules()
			if len(rules) > 0 {
				desc.Add(jsonutils.NewString(rules), "admin_security_rules")
			}
		}
	*/

	extraOptions := self.getExtraOptions()
	if extraOptions != nil {
		desc.Add(extraOptions, "extra_options")
	}

	kvmOptions := self.getKvmOptions()
	if len(kvmOptions) > 0 {
		desc.Add(jsonutils.NewString(kvmOptions), "kvm")
	}

	zone := self.getZone()
	if zone != nil {
		desc.Add(jsonutils.NewString(zone.Id), "zone_id")
		desc.Add(jsonutils.NewString(zone.Name), "zone")
	}

	os := self.GetOS()
	if len(os) > 0 {
		desc.Add(jsonutils.NewString(os), "os_name")
	}

	meta, _ := self.GetAllMetadata(nil)
	desc.Add(jsonutils.Marshal(meta), "metadata")

	userData := meta["user_data"]
	if len(userData) > 0 {
		desc.Add(jsonutils.NewString(userData), "user_data")
	}

	if self.PendingDeleted {
		desc.Add(jsonutils.JSONTrue, "pending_deleted")
	} else {
		desc.Add(jsonutils.JSONFalse, "pending_deleted")
	}

	return desc
}

func (self *SGuest) GetJsonDescAtBaremetal(ctx context.Context, host *SHost) *jsonutils.JSONDict {
	desc := jsonutils.NewDict()

	desc.Add(jsonutils.NewString(self.Name), "name")
	if len(self.Description) > 0 {
		desc.Add(jsonutils.NewString(self.Description), "description")
	}
	desc.Add(jsonutils.NewString(self.Id), "uuid")
	desc.Add(jsonutils.NewInt(int64(self.VmemSize)), "mem")
	desc.Add(jsonutils.NewInt(int64(self.VcpuCount)), "cpu")
	diskConf := host.getDiskConfig()
	if diskConf != nil {
		desc.Add(diskConf, "disk_config")
	}

	jsonNics := make([]jsonutils.JSONObject, 0)
	jsonStandbyNics := make([]jsonutils.JSONObject, 0)

	netifs := host.GetNetInterfaces()
	domain := options.Options.DNSDomain

	if netifs != nil && len(netifs) > 0 {
		for _, nic := range netifs {
			nicDesc := nic.getServerJsonDesc()
			if nicDesc.Contains("ip") {
				jsonNics = append(jsonNics, nicDesc)
				nicDomain, _ := nicDesc.GetString("domain")
				if len(nicDomain) > 0 && len(domain) == 0 {
					domain = nicDomain
				}
			} else {
				jsonStandbyNics = append(jsonStandbyNics, nicDesc)
			}
		}
	}
	desc.Add(jsonutils.NewArray(jsonNics...), "nics")
	desc.Add(jsonutils.NewArray(jsonStandbyNics...), "nics_standby")
	desc.Add(jsonutils.NewString(domain), "domain")

	jsonDisks := make([]jsonutils.JSONObject, 0)
	disks := self.GetDisks()
	if disks != nil && len(disks) > 0 {
		for _, disk := range disks {
			diskDesc := disk.GetJsonDescAtHost(host)
			jsonDisks = append(jsonDisks, diskDesc)
		}
	}
	desc.Add(jsonutils.NewArray(jsonDisks...), "disks")

	tc, _ := self.GetTenantCache(ctx)
	if tc != nil {
		desc.Add(jsonutils.NewString(tc.GetName()), "tenant")
	}

	desc.Add(jsonutils.NewString(self.ProjectId), "tenant_id")

	keypair := self.getKeypair()
	if keypair != nil {
		desc.Add(jsonutils.NewString(keypair.Name), "keypair")
		desc.Add(jsonutils.NewString(keypair.PublicKey), "pubkey")
	}

	netRoles := self.getNetworkRoles()
	if netRoles != nil && len(netRoles) > 0 {
		desc.Add(jsonutils.NewStringArray(netRoles), "network_roles")
	}

	rules := self.getSecurityRules()
	if len(rules) > 0 {
		desc.Add(jsonutils.NewString(rules), "security_rules")
	}
	rules = self.getAdminSecurityRules()
	if len(rules) > 0 {
		desc.Add(jsonutils.NewString(rules), "admin_security_rules")
	}

	zone := self.getZone()
	if zone != nil {
		desc.Add(jsonutils.NewString(zone.Id), "zone_id")
		desc.Add(jsonutils.NewString(zone.Name), "zone")
	}

	os := self.GetOS()
	if len(os) > 0 {
		desc.Add(jsonutils.NewString(os), "os_name")
	}

	meta, _ := self.GetAllMetadata(nil)
	desc.Add(jsonutils.Marshal(meta), "metadata")

	userData := meta["user_data"]
	if len(userData) > 0 {
		desc.Add(jsonutils.NewString(userData), "user_data")
	}

	if self.PendingDeleted {
		desc.Add(jsonutils.JSONTrue, "pending_deleted")
	} else {
		desc.Add(jsonutils.JSONFalse, "pending_deleted")
	}

	return desc
}

func (self *SGuest) getNetworkRoles() []string {
	key := db.Metadata.GetSysadminKey("network_role")
	roleStr := self.GetMetadata(key, auth.AdminCredential())
	if len(roleStr) > 0 {
		return strings.Split(roleStr, ",")
	}
	return nil
}

func (manager *SGuestManager) FetchGuestById(guestId string) *SGuest {
	guest, err := manager.FetchById(guestId)
	if err != nil {
		log.Errorf("FetchById fail %s", err)
		return nil
	}
	return guest.(*SGuest)
}

func (self *SGuest) saveOsType(osType string) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.OsType = osType
		return nil
	})
	return err
}

func (self *SGuest) SaveDeployInfo(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) {
	info := make(map[string]interface{})
	if data.Contains("os") {
		osName, _ := data.GetString("os")
		self.saveOsType(osName)
		info["os_name"] = osName
	}
	if data.Contains("account") {
		account, _ := data.GetString("account")
		info["login_account"] = account
		if data.Contains("key") {
			key, _ := data.GetString("key")
			info["login_key"] = key
			info["login_key_timestamp"] = timeutils.UtcNow()
		} else {
			info["login_key"] = "none"
			info["login_key_timestamp"] = "none"
		}
	}
	if data.Contains("distro") {
		dist, _ := data.GetString("distro")
		info["os_distribution"] = dist
	}
	if data.Contains("version") {
		ver, _ := data.GetString("version")
		info["os_version"] = ver
	}
	if data.Contains("arch") {
		arch, _ := data.GetString("arch")
		info["os_arch"] = arch
	}
	if data.Contains("language") {
		lang, _ := data.GetString("language")
		info["os_language"] = lang
	}
	self.SetAllMetadata(ctx, info, userCred)
}

func (self *SGuest) isAllDisksReady() bool {
	ready := true
	disks := self.GetDisks()
	if disks == nil || len(disks) == 0 {
		log.Errorf("No valid disks")
		return false
	}
	for i := 0; i < len(disks); i += 1 {
		disk := disks[i].GetDisk()
		if !(disk.isReady() || disk.Status == DISK_START_MIGRATE) {
			ready = false
			break
		}
	}
	return ready
}

func (self *SGuest) AllowPerformStart(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SGuest) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{VM_READY, VM_START_FAILED, VM_SAVE_DISK_FAILED, VM_SUSPEND}) {
		if self.isAllDisksReady() {
			var kwargs *jsonutils.JSONDict
			if data == nil {
				kwargs = data.(*jsonutils.JSONDict)
			}
			err := self.GetDriver().PerformStart(ctx, userCred, self, kwargs)
			return nil, err
		} else {
			msg := "Some disk not ready"
			return nil, httperrors.NewResourceNotReadyError(msg)
		}
	} else {
		return nil, httperrors.NewInvalidStatusError("Cannot do start server in status %s", self.Status)
	}
}

func (self *SGuest) AllowPerformStop(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SGuest) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	isForce := jsonutils.QueryBoolean(data, "is_force", false)
	if utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_STOP_FAILED}) || (isForce && self.Status == VM_STOPPING) {
		return nil, self.StartGuestStopTask(ctx, userCred, isForce, "")
	} else {
		return nil, httperrors.NewInvalidStatusError("Cannot do start server in status %s", self.Status)
	}
}

/*

TODO

def start_guest_sched_start_task(self, user_cred, data=None,
                                            parent_task_id=None):
        from clouds.models.tasks import Tasks
        from clouds.tasks import worker
        from clouds.tasks.guests import GuestSchedStartTask
        kwargs = {}
        kwargs['guest_status'] = self.status
        if data is not None:
            kwargs['params'] = data
        self.set_status(self.VM_SCHEDULE)
        task = Tasks.new_task(GuestSchedStartTask, self, user_cred, kwargs,
                                    parent_task_id=parent_task_id)
        worker.get_manager().exec_task(task)

*/

func (self *SGuest) AllowGetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SGuest) GetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{VM_RUNNING, VM_SNAPSHOT_STREAM}) {
		host := self.GetHost()
		if host == nil {
			return nil, httperrors.NewInternalServerError("Host missing")
		}
		retval, err := self.GetDriver().GetGuestVncInfo(userCred, self, host)
		if err != nil {
			return nil, err
		}
		retval.Add(jsonutils.NewString(self.Id), "id")
		return retval, nil
	} else {
		return jsonutils.NewDict(), nil
	}
}

func (self *SGuest) GetKeypairPublicKey() string {
	keypair := self.getKeypair()
	if keypair != nil {
		return keypair.PublicKey
	}
	return ""
}

func (manager *SGuestManager) GetIpInProjectWithName(projectId, name string, isExitOnly bool) []string {
	guestnics := GuestnetworkManager.Query().SubQuery()
	guests := manager.Query().SubQuery()
	networks := NetworkManager.Query().SubQuery()
	q := guestnics.Query(guestnics.Field("ip_addr")).Join(guests,
		sqlchemy.AND(
			sqlchemy.Equals(guests.Field("id"), guestnics.Field("guest_id")),
			sqlchemy.OR(sqlchemy.IsNull(guests.Field("pending_deleted")),
				sqlchemy.IsFalse(guests.Field("pending_deleted"))),
			sqlchemy.IsFalse(guests.Field("deleted")))).
		Join(networks, sqlchemy.AND(sqlchemy.Equals(networks.Field("id"), guestnics.Field("network_id")),
			sqlchemy.IsFalse(networks.Field("deleted")))).
		Filter(sqlchemy.Equals(guests.Field("name"), name)).
		Filter(sqlchemy.NotEquals(guestnics.Field("ip_addr"), "")).
		Filter(sqlchemy.IsNotNull(guestnics.Field("ip_addr"))).
		Filter(sqlchemy.IsNotNull(networks.Field("guest_gateway")))
	ips := make([]string, 0)
	rows, err := q.Rows()
	if err != nil {
		log.Errorf("Get guest ip with name query err: %v", err)
		return ips
	}
	for rows.Next() {
		var ip string
		err = rows.Scan(&ip)
		if err != nil {
			log.Errorf("Get guest ip with name scan err: %v", err)
			return ips
		}
		ips = append(ips, ip)
	}
	return manager.getIpsByExit(ips, isExitOnly)
}

func (manager *SGuestManager) getIpsByExit(ips []string, isExitOnly bool) []string {
	intRet := make([]string, 0)
	extRet := make([]string, 0)
	for _, ip := range ips {
		addr, _ := netutils.NewIPV4Addr(ip)
		if netutils.IsExitAddress(addr) {
			extRet = append(extRet, ip)
			continue
		}
		intRet = append(intRet, ip)
	}
	if isExitOnly {
		return extRet
	} else if len(intRet) > 0 {
		return intRet
	}
	return extRet
}
