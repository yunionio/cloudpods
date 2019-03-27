package models

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/compute/sshkeys"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

const (
	VM_INIT            = api.VM_INIT
	VM_UNKNOWN         = api.VM_UNKNOWN
	VM_SCHEDULE        = api.VM_SCHEDULE
	VM_SCHEDULE_FAILED = api.VM_SCHEDULE_FAILED
	VM_CREATE_NETWORK  = api.VM_CREATE_NETWORK
	VM_NETWORK_FAILED  = api.VM_NETWORK_FAILED
	VM_DEVICE_FAILED   = api.VM_DEVICE_FAILED
	VM_CREATE_FAILED   = api.VM_CREATE_FAILED
	VM_CREATE_DISK     = api.VM_CREATE_DISK
	VM_DISK_FAILED     = api.VM_DISK_FAILED
	VM_START_DEPLOY    = api.VM_START_DEPLOY
	VM_DEPLOYING       = api.VM_DEPLOYING
	VM_DEPLOY_FAILED   = api.VM_DEPLOY_FAILED
	VM_READY           = api.VM_READY
	VM_START_START     = api.VM_START_START
	VM_STARTING        = api.VM_STARTING
	VM_START_FAILED    = api.VM_START_FAILED // # = ready
	VM_RUNNING         = api.VM_RUNNING
	VM_START_STOP      = api.VM_START_STOP
	VM_STOPPING        = api.VM_STOPPING
	VM_STOP_FAILED     = api.VM_STOP_FAILED // # = running

	VM_BACKUP_STARTING         = api.VM_BACKUP_STARTING
	VM_BACKUP_CREATING         = api.VM_BACKUP_CREATING
	VM_BACKUP_CREATE_FAILED    = api.VM_BACKUP_CREATE_FAILED
	VM_DEPLOYING_BACKUP        = api.VM_DEPLOYING_BACKUP
	VM_DEPLOYING_BACKUP_FAILED = api.VM_DEPLOYING_BACKUP_FAILED
	VM_DELETING_BACKUP         = api.VM_DELETING_BACKUP
	VM_BACKUP_DELETE_FAILED    = api.VM_BACKUP_DELETE_FAILED
	VM_SWITCH_TO_BACKUP        = api.VM_SWITCH_TO_BACKUP
	VM_SWITCH_TO_BACKUP_FAILED = api.VM_SWITCH_TO_BACKUP_FAILED

	VM_ATTACH_DISK_FAILED = api.VM_ATTACH_DISK_FAILED
	VM_DETACH_DISK_FAILED = api.VM_DETACH_DISK_FAILED

	VM_START_SUSPEND  = api.VM_START_SUSPEND
	VM_SUSPENDING     = api.VM_SUSPENDING
	VM_SUSPEND        = api.VM_SUSPEND
	VM_SUSPEND_FAILED = api.VM_SUSPEND_FAILED

	VM_START_DELETE = api.VM_START_DELETE
	VM_DELETE_FAIL  = api.VM_DELETE_FAIL
	VM_DELETING     = api.VM_DELETING

	VM_DEALLOCATED = api.VM_DEALLOCATED

	VM_START_MIGRATE  = api.VM_START_MIGRATE
	VM_MIGRATING      = api.VM_MIGRATING
	VM_MIGRATE_FAILED = api.VM_MIGRATE_FAILED

	VM_CHANGE_FLAVOR      = api.VM_CHANGE_FLAVOR
	VM_CHANGE_FLAVOR_FAIL = api.VM_CHANGE_FLAVOR_FAIL
	VM_REBUILD_ROOT       = api.VM_REBUILD_ROOT
	VM_REBUILD_ROOT_FAIL  = api.VM_REBUILD_ROOT_FAIL

	VM_START_SNAPSHOT  = api.VM_START_SNAPSHOT
	VM_SNAPSHOT        = api.VM_SNAPSHOT
	VM_SNAPSHOT_DELETE = api.VM_SNAPSHOT_DELETE
	VM_BLOCK_STREAM    = api.VM_BLOCK_STREAM
	VM_MIRROR_FAIL     = api.VM_MIRROR_FAIL
	VM_SNAPSHOT_SUCC   = api.VM_SNAPSHOT_SUCC
	VM_SNAPSHOT_FAILED = api.VM_SNAPSHOT_FAILED

	VM_SYNCING_STATUS = api.VM_SYNCING_STATUS
	VM_SYNC_CONFIG    = api.VM_SYNC_CONFIG
	VM_SYNC_FAIL      = api.VM_SYNC_FAIL

	VM_RESIZE_DISK        = api.VM_RESIZE_DISK
	VM_RESIZE_DISK_FAILED = api.VM_RESIZE_DISK_FAILED
	VM_START_SAVE_DISK    = api.VM_START_SAVE_DISK
	VM_SAVE_DISK          = api.VM_SAVE_DISK
	VM_SAVE_DISK_FAILED   = api.VM_SAVE_DISK_FAILED

	VM_RESTORING_SNAPSHOT = api.VM_RESTORING_SNAPSHOT
	VM_RESTORE_DISK       = api.VM_RESTORE_DISK
	VM_RESTORE_STATE      = api.VM_RESTORE_STATE
	VM_RESTORE_FAILED     = api.VM_RESTORE_FAILED

	VM_ASSOCIATE_EIP         = api.VM_ASSOCIATE_EIP
	VM_ASSOCIATE_EIP_FAILED  = api.VM_ASSOCIATE_EIP_FAILED
	VM_DISSOCIATE_EIP        = api.VM_DISSOCIATE_EIP
	VM_DISSOCIATE_EIP_FAILED = api.VM_DISSOCIATE_EIP_FAILED

	VM_REMOVE_STATEFILE = api.VM_REMOVE_STATEFILE

	VM_ADMIN = api.VM_ADMIN

	SHUTDOWN_STOP      = api.SHUTDOWN_STOP
	SHUTDOWN_TERMINATE = api.SHUTDOWN_TERMINATE

	HYPERVISOR_KVM       = api.HYPERVISOR_KVM
	HYPERVISOR_CONTAINER = api.HYPERVISOR_CONTAINER
	HYPERVISOR_BAREMETAL = api.HYPERVISOR_BAREMETAL
	HYPERVISOR_ESXI      = api.HYPERVISOR_ESXI
	HYPERVISOR_HYPERV    = api.HYPERVISOR_HYPERV
	HYPERVISOR_XEN       = api.HYPERVISOR_XEN

	HYPERVISOR_ALIYUN    = api.HYPERVISOR_ALIYUN
	HYPERVISOR_QCLOUD    = api.HYPERVISOR_QCLOUD
	HYPERVISOR_AZURE     = api.HYPERVISOR_AZURE
	HYPERVISOR_AWS       = api.HYPERVISOR_AWS
	HYPERVISOR_HUAWEI    = api.HYPERVISOR_HUAWEI
	HYPERVISOR_OPENSTACK = api.HYPERVISOR_OPENSTACK
	HYPERVISOR_UCLOUD    = api.HYPERVISOR_UCLOUD

	//	HYPERVISOR_DEFAULT = HYPERVISOR_KVM
	HYPERVISOR_DEFAULT = HYPERVISOR_KVM
)

var VM_RUNNING_STATUS = api.VM_RUNNING_STATUS
var VM_CREATING_STATUS = api.VM_CREATING_STATUS

var HYPERVISORS = api.HYPERVISORS

var PUBLIC_CLOUD_HYPERVISORS = api.PUBLIC_CLOUD_HYPERVISORS

// var HYPERVISORS = []string{HYPERVISOR_ALIYUN}

var HYPERVISOR_HOSTTYPE = api.HYPERVISOR_HOSTTYPE

var HOSTTYPE_HYPERVISOR = api.HOSTTYPE_HYPERVISOR

type SGuestManager struct {
	db.SVirtualResourceBaseManager
}

var GuestManager *SGuestManager

func init() {
	GuestManager = &SGuestManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SGuest{},
			"guests_tbl",
			"server",
			"servers",
		),
	}
	GuestManager.SetAlias("guest", "guests")
}

type SGuest struct {
	db.SVirtualResourceBase

	SBillingResourceBase

	VcpuCount int8 `nullable:"false" default:"1" list:"user" create:"optional"` // Column(TINYINT, nullable=False, default=1)
	VmemSize  int  `nullable:"false" list:"user" create:"required"`             // Column(Integer, nullable=False)

	BootOrder string `width:"8" charset:"ascii" nullable:"true" default:"cdn" list:"user" update:"user" create:"optional"` // Column(VARCHAR(8, charset='ascii'), nullable=True, default='cdn')

	DisableDelete    tristate.TriState `nullable:"false" default:"true" list:"user" update:"user" create:"optional"`           // Column(Boolean, nullable=False, default=True)
	ShutdownBehavior string            `width:"16" charset:"ascii" default:"stop" list:"user" update:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), default=SHUTDOWN_STOP)

	KeypairId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	HostId       string `width:"36" charset:"ascii" nullable:"true" list:"admin" get:"admin"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	BackupHostId string `width:"36" charset:"ascii" nullable:"true" list:"admin" get:"admin"`

	Vga     string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Vdi     string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Machine string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Bios    string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	OsType  string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	FlavorId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	SecgrpId      string `width:"36" charset:"ascii" nullable:"true" get:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
	AdminSecgrpId string `width:"36" charset:"ascii" nullable:"true" get:"admin"`                  // Column(VARCHAR(36, charset='ascii'), nullable=True)

	Hypervisor string `width:"16" charset:"ascii" nullable:"false" default:"kvm" list:"user" create:"required"` // Column(VARCHAR(16, charset='ascii'), nullable=False, default=HYPERVISOR_DEFAULT)

	InstanceType string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

func (manager *SGuestManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	if query.Contains("host") || query.Contains("wire") || query.Contains("zone") {
		if !db.IsAdminAllowList(userCred, manager) {
			return false
		}
	}
	return manager.SVirtualResourceBaseManager.AllowListItems(ctx, userCred, query)
}

func (manager *SGuestManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	queryDict, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("invalid querystring format")
	}

	var err error
	q, err = managedResourceFilterByAccount(q, query, "host_id", func() *sqlchemy.SQuery {
		hosts := HostManager.Query().SubQuery()
		return hosts.Query(hosts.Field("id"))
	})
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "host_id", func() *sqlchemy.SQuery {
		hosts := HostManager.Query().SubQuery()
		return hosts.Query(hosts.Field("id"))
	})

	billingTypeStr, _ := queryDict.GetString("billing_type")
	if len(billingTypeStr) > 0 {
		if billingTypeStr == BILLING_TYPE_POSTPAID {
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.IsNullOrEmpty(q.Field("billing_type")),
					sqlchemy.Equals(q.Field("billing_type"), billingTypeStr),
				),
			)
		} else {
			q = q.Equals("billing_type", billingTypeStr)
		}
		queryDict.Remove("billing_type")
	}

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
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

	resourceTypeStr := jsonutils.GetAnyString(queryDict, []string{"resource_type"})
	if len(resourceTypeStr) > 0 {
		hosts := HostManager.Query().SubQuery()
		subq := hosts.Query(hosts.Field("id"))
		switch resourceTypeStr {
		case HostResourceTypeShared:
			subq = subq.Filter(
				sqlchemy.OR(
					sqlchemy.IsNullOrEmpty(hosts.Field("resource_type")),
					sqlchemy.Equals(hosts.Field("resource_type"), resourceTypeStr),
				),
			)
		default:
			subq = subq.Equals("resource_type", resourceTypeStr)
		}

		q = q.In("host_id", subq.SubQuery())
	}

	hostFilter, _ := queryDict.GetString("host")
	if len(hostFilter) > 0 {
		host, _ := HostManager.FetchByIdOrName(nil, hostFilter)
		if host == nil {
			return nil, httperrors.NewResourceNotFoundError("host %s not found", hostFilter)
		}
		if jsonutils.QueryBoolean(queryDict, "get_backup_guests_on_host", false) {
			q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("host_id"), host.GetId()),
				sqlchemy.Equals(q.Field("backup_host_id"), host.GetId())))
		} else {
			q = q.Equals("host_id", host.GetId())
		}
	}

	secgrpFilter, _ := queryDict.GetString("secgroup")
	if len(secgrpFilter) > 0 {
		var notIn = false
		// HACK FOR NOT IN SECGROUP
		if strings.HasPrefix(secgrpFilter, "!") {
			secgrpFilter = secgrpFilter[1:]
			notIn = true
		}
		secgrp, _ := SecurityGroupManager.FetchByIdOrName(userCred, secgrpFilter)
		if secgrp == nil {
			return nil, httperrors.NewResourceNotFoundError("secgroup %s not found", secgrpFilter)
		}

		if notIn {
			filter1 := sqlchemy.NotIn(q.Field("id"),
				GuestsecgroupManager.Query("guest_id").Equals("secgroup_id", secgrp.GetId()).SubQuery())
			filter2 := sqlchemy.NotEquals(q.Field("secgrp_id"), secgrp.GetId())
			q = q.Filter(sqlchemy.AND(filter1, filter2))
		} else {
			filter1 := sqlchemy.In(q.Field("id"),
				GuestsecgroupManager.Query("guest_id").Equals("secgroup_id", secgrp.GetId()).SubQuery())
			filter2 := sqlchemy.Equals(q.Field("secgrp_id"), secgrp.GetId())
			q = q.Filter(sqlchemy.OR(filter1, filter2))
		}
	}

	zoneFilter, _ := queryDict.GetString("zone")
	if len(zoneFilter) > 0 {
		zone, _ := ZoneManager.FetchByIdOrName(nil, zoneFilter)
		if zone == nil {
			return nil, httperrors.NewResourceNotFoundError("zone %s not found", zoneFilter)
		}
		hostTable := HostManager.Query().SubQuery()
		zoneTable := ZoneManager.Query().SubQuery()
		sq := hostTable.Query(hostTable.Field("id")).Join(zoneTable,
			sqlchemy.Equals(zoneTable.Field("id"), hostTable.Field("zone_id"))).Filter(sqlchemy.Equals(zoneTable.Field("id"), zone.GetId())).SubQuery()
		q = q.In("host_id", sq)
	}

	wireFilter, _ := queryDict.GetString("wire")
	if len(wireFilter) > 0 {
		wire, _ := WireManager.FetchByIdOrName(nil, wireFilter)
		if wire == nil {
			return nil, httperrors.NewResourceNotFoundError("wire %s not found", wireFilter)
		}
		hostTable := HostManager.Query().SubQuery()
		hostWire := HostwireManager.Query().SubQuery()
		sq := hostTable.Query(hostTable.Field("id")).Join(hostWire, sqlchemy.Equals(hostWire.Field("host_id"), hostTable.Field("id"))).Filter(sqlchemy.Equals(hostWire.Field("wire_id"), wire.GetId())).SubQuery()
		q = q.In("host_id", sq)
	}

	networkFilter, _ := queryDict.GetString("network")
	if len(networkFilter) > 0 {
		netI, _ := NetworkManager.FetchByIdOrName(userCred, networkFilter)
		if netI == nil {
			return nil, httperrors.NewResourceNotFoundError("network %s not found", networkFilter)
		}
		net := netI.(*SNetwork)
		hostTable := HostManager.Query().SubQuery()
		hostWire := HostwireManager.Query().SubQuery()
		sq := hostTable.Query(hostTable.Field("id")).Join(hostWire,
			sqlchemy.Equals(hostWire.Field("host_id"), hostTable.Field("id"))).Filter(sqlchemy.Equals(hostWire.Field("wire_id"), net.WireId)).SubQuery()
		q = q.In("host_id", sq)
	}

	vpcFilter, err := queryDict.GetString("vpc")
	if err == nil {
		IVpc, err := VpcManager.FetchByIdOrName(userCred, vpcFilter)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Vpc %s not found", vpcFilter)
		}
		vpc := IVpc.(*SVpc)
		guestnetwork := GuestnetworkManager.Query().SubQuery()
		network := NetworkManager.Query().SubQuery()
		wire := WireManager.Query().SubQuery()
		sq := guestnetwork.Query(guestnetwork.Field("guest_id")).Join(network,
			sqlchemy.Equals(guestnetwork.Field("network_id"), network.Field("id"))).
			Join(wire, sqlchemy.Equals(network.Field("wire_id"), wire.Field("id"))).
			Filter(sqlchemy.Equals(wire.Field("vpc_id"), vpc.Id)).SubQuery()
		q = q.In("id", sq)
	}

	diskFilter, _ := queryDict.GetString("disk")
	if len(diskFilter) > 0 {
		diskI, _ := DiskManager.FetchByIdOrName(userCred, diskFilter)
		if diskI == nil {
			return nil, httperrors.NewResourceNotFoundError("disk %s not found", diskFilter)
		}
		disk := diskI.(*SDisk)
		guestdisks := GuestdiskManager.Query().SubQuery()
		count := guestdisks.Query().Equals("disk_id", disk.Id).Count()
		if count > 0 {
			sgq := guestdisks.Query(guestdisks.Field("guest_id")).Equals("disk_id", disk.Id).SubQuery()
			q = q.Filter(sqlchemy.In(q.Field("id"), sgq))
		} else {
			hosts := HostManager.Query().SubQuery()
			hoststorages := HoststorageManager.Query().SubQuery()
			storages := StorageManager.Query().SubQuery()
			sq := hosts.Query(hosts.Field("id")).
				Join(hoststorages, sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id"))).
				Join(storages, sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id"))).
				Filter(sqlchemy.Equals(storages.Field("id"), disk.StorageId)).SubQuery()
			q = q.In("host_id", sq)
		}
	}

	regionFilter, _ := queryDict.GetString("region")
	if len(regionFilter) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(userCred, regionFilter)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError("cloud region %s not found", regionFilter)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		hosts := HostManager.Query().SubQuery()
		zones := ZoneManager.Query().SubQuery()
		sq := hosts.Query(hosts.Field("id"))
		sq = sq.Join(zones, sqlchemy.Equals(hosts.Field("zone_id"), zones.Field("id")))
		sq = sq.Filter(sqlchemy.Equals(zones.Field("cloudregion_id"), regionObj.GetId()))
		q = q.In("host_id", sq)
	}

	withEip, _ := queryDict.GetString("with_eip")
	withoutEip, _ := queryDict.GetString("without_eip")
	if len(withEip) > 0 || len(withoutEip) > 0 {
		eips := ElasticipManager.Query().SubQuery()
		sq := eips.Query(eips.Field("associate_id")).Equals("associate_type", EIP_ASSOCIATE_TYPE_SERVER)
		sq = sq.IsNotNull("associate_id").IsNotEmpty("associate_id")

		if utils.ToBool(withEip) {
			q = q.In("id", sq)
		} else if utils.ToBool(withoutEip) {
			q = q.NotIn("id", sq)
		}
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

	orderByDisk, _ := queryDict.GetString("order_by_disk")
	if orderByDisk == "asc" {
		guestdisks := GuestdiskManager.Query().SubQuery()
		disks := DiskManager.Query().SubQuery()
		q.AppendField(sqlchemy.SUM("disks_size", disks.Field("disk_size")))
		q = q.Join(guestdisks, sqlchemy.Equals(q.Field("id"), guestdisks.Field("guest_id"))).
			Join(disks, sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id"))).
			Asc(q.Field("disks_size")).GroupBy(q.Field("id"))
	} else if orderByDisk == "desc" {
		guestdisks := GuestdiskManager.Query().SubQuery()
		disks := DiskManager.Query().SubQuery()
		q.AppendField(sqlchemy.SUM("disks_size", disks.Field("disk_size")))
		q = q.Join(guestdisks, sqlchemy.Equals(q.Field("id"), guestdisks.Field("guest_id"))).
			Join(disks, sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id"))).
			Desc(q.Field("disks_size")).GroupBy(q.Field("id"))
	}

	orderByHost, _ := queryDict.GetString("order_by_host")
	if orderByHost == "asc" {
		hosts := HostManager.Query().SubQuery()
		q = q.Join(hosts, sqlchemy.Equals(q.Field("host_id"), hosts.Field("id"))).
			Asc(hosts.Field("name"))
	} else if orderByHost == "desc" {
		hosts := HostManager.Query().SubQuery()
		q = q.Join(hosts, sqlchemy.Equals(q.Field("host_id"), hosts.Field("id"))).
			Desc(hosts.Field("name"))
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

func (guest *SGuest) validateDeleteCondition(ctx context.Context, isPurge bool) error {
	if guest.DisableDelete.IsTrue() {
		return httperrors.NewInvalidStatusError("Virtual server is locked, cannot delete")
	}
	if !isPurge && guest.IsValidPrePaid() {
		return httperrors.NewForbiddenError("not allow to delete prepaid server in valid status")
	}
	return guest.SVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (guest *SGuest) ValidatePurgeCondition(ctx context.Context) error {
	return guest.validateDeleteCondition(ctx, true)
}

func (guest *SGuest) ValidateDeleteCondition(ctx context.Context) error {
	return guest.validateDeleteCondition(ctx, false)
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

func (guest *SGuest) GetNetworksQuery(netId string) *sqlchemy.SQuery {
	q := GuestnetworkManager.Query().Equals("guest_id", guest.Id)
	if len(netId) > 0 {
		q = q.Equals("network_id", netId)
	}
	return q
}

func (guest *SGuest) NetworkCount() int {
	return guest.GetNetworksQuery("").Count()
}

func (guest *SGuest) GetNetworks(netId string) ([]SGuestnetwork, error) {
	guestnics := make([]SGuestnetwork, 0)
	q := guest.GetNetworksQuery(netId).Asc("index")
	err := db.FetchModelObjects(GuestnetworkManager, q, &guestnics)
	if err != nil {
		log.Errorf("GetNetworks error: %s", err)
		return nil, err
	}
	return guestnics, nil
}

func (guest *SGuest) getGuestnetworkByIpOrMac(ipAddr string, macAddr string) (*SGuestnetwork, error) {
	q := guest.GetNetworksQuery("")
	if len(ipAddr) > 0 {
		q = q.Equals("ip_addr", ipAddr)
	}
	if len(macAddr) > 0 {
		q = q.Equals("mac_addr", macAddr)
	}

	guestnic := SGuestnetwork{}
	err := q.First(&guestnic)
	if err != nil {
		return nil, err
	}
	guestnic.SetModelManager(GuestnetworkManager)
	return &guestnic, nil
}

func (guest *SGuest) GetGuestnetworkByIp(ipAddr string) (*SGuestnetwork, error) {
	return guest.getGuestnetworkByIpOrMac(ipAddr, "")
}

func (guest *SGuest) GetGuestnetworkByMac(macAddr string) (*SGuestnetwork, error) {
	return guest.getGuestnetworkByIpOrMac("", macAddr)
}

func (guest *SGuest) IsNetworkAllocated() bool {
	guestnics, err := guest.GetNetworks("")
	if err != nil {
		return false
	}
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

func (guest *SGuest) SetHostId(userCred mcclient.TokenCredential, hostId string) error {
	if guest.HostId != hostId {
		diff, err := db.Update(guest, func() error {
			guest.HostId = hostId
			return nil
		})
		if err != nil {
			return err
		}
		db.OpsLog.LogEvent(guest, db.ACT_UPDATE, diff, userCred)
	}
	return nil
}

func (guest *SGuest) SetHostIdWithBackup(userCred mcclient.TokenCredential, master, slave string) error {
	diff, err := db.Update(guest, func() error {
		guest.HostId = master
		guest.BackupHostId = slave
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(guest, db.ACT_UPDATE, diff, userCred)
	return err
}

func (guest *SGuest) ValidateResizeDisk(disk *SDisk, storage *SStorage) error {
	return guest.GetDriver().ValidateResizeDisk(guest, disk, storage)
}

func ValidateMemData(vmemSize int, driver IGuestDriver) (int, error) {
	if vmemSize > 0 {
		maxVmemGb := driver.GetMaxVMemSizeGB()
		if vmemSize < 8 || vmemSize > maxVmemGb*1024 {
			return 0, httperrors.NewInputParameterError("Memory size must be 8MB ~ %d GB", maxVmemGb)
		}
	}
	return vmemSize, nil
}

func ValidateCpuData(vcpuCount int, driver IGuestDriver) (int, error) {
	maxVcpuCount := driver.GetMaxVCpuCount()
	if vcpuCount < 1 || vcpuCount > maxVcpuCount {
		return 0, httperrors.NewInputParameterError("CPU core count must be 1 ~ %d", maxVcpuCount)
	}
	return vcpuCount, nil
}

func ValidateMemCpuData(vmemSize, vcpuCount int, hypervisor string) (int, int, error) {
	if len(hypervisor) == 0 {
		hypervisor = HYPERVISOR_DEFAULT
	}
	driver := GetDriver(hypervisor)

	var err error
	vmemSize, err = ValidateMemData(vmemSize, driver)
	if err != nil {
		return 0, 0, err
	}

	vcpuCount, err = ValidateCpuData(vcpuCount, driver)
	if err != nil {
		return 0, 0, err
	}
	return vmemSize, vcpuCount, nil
}

func (self *SGuest) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var err error
	var vmemSize int
	var vcpuCount int

	driver := GetDriver(self.Hypervisor)

	if memSize, _ := data.Int("vmem_size"); memSize != 0 {
		vmemSize, err = ValidateMemData(int(memSize), driver)
		if err != nil {
			return nil, err
		}
	}
	if cpuCount, _ := data.Int("vcpu_count"); cpuCount != 0 {
		vcpuCount, err = ValidateCpuData(int(cpuCount), driver)
		if err != nil {
			return nil, err
		}
	}

	if vmemSize > 0 || vcpuCount > 0 {
		if !utils.IsInStringArray(self.Status, []string{VM_READY}) && self.GetHypervisor() != HYPERVISOR_CONTAINER {
			return nil, httperrors.NewInvalidStatusError("Cannot modify Memory and CPU in status %s", self.Status)
		}
		if self.GetHypervisor() == HYPERVISOR_BAREMETAL {
			return nil, httperrors.NewInputParameterError("Cannot modify memory for baremetal")
		}
	}

	if vmemSize > 0 {
		data.Add(jsonutils.NewInt(int64(vmemSize)), "vmem_size")
	}
	if vcpuCount > 0 {
		data.Add(jsonutils.NewInt(int64(vcpuCount)), "vcpu_count")
	}

	data, err = self.GetDriver().ValidateUpdateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}

	err = self.checkUpdateQuota(ctx, userCred, vcpuCount, vmemSize)
	if err != nil {
		return nil, httperrors.NewOutOfQuotaError(err.Error())
	}

	if data.Contains("name") {
		if name, _ := data.GetString("name"); len(name) < 2 {
			return nil, httperrors.NewInputParameterError("name is too short")
		}
	}
	return self.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SGuestManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// TODO: 定义 api.ServerCreateInput 的 Unmarshal 函数，直接通过 data.Unmarshal(input) 解析参数
	input, err := cmdline.FetchServerCreateInputByJSON(data)
	if err != nil {
		return nil, err
	}
	resetPassword := true
	if input.ResetPassword != nil {
		resetPassword = *input.ResetPassword
	}

	passwd := input.Password
	if resetPassword && len(passwd) > 0 {
		if !seclib2.MeetComplxity(passwd) {
			return nil, httperrors.NewWeakPasswordError()
		}
	}

	var hypervisor string
	// var rootStorageType string
	var osProf osprofile.SOSProfile
	hypervisor = input.Hypervisor
	if hypervisor != HYPERVISOR_CONTAINER {
		if len(input.Disks) == 0 {
			return nil, httperrors.NewInputParameterError("No disk information provided")
		}
		diskConfig := input.Disks[0]
		diskConfig, err = parseDiskInfo(ctx, userCred, diskConfig)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Invalid root image: %s", err)
		}
		if len(diskConfig.SnapshotId) > 0 && diskConfig.DiskType != DISK_TYPE_SYS {
			return nil, httperrors.NewBadRequestError("Snapshot error: disk index 0 but disk type is %s", diskConfig.DiskType)
		}

		if len(diskConfig.ImageId) == 0 && len(diskConfig.SnapshotId) == 0 && !data.Contains("cdrom") {
			return nil, httperrors.NewBadRequestError("Miss operating system???")
		}

		// if len(diskConfig.Backend) == 0 {
		// 	diskConfig.Backend = STORAGE_LOCAL
		// }
		// rootStorageType = diskConfig.Backend

		input.Disks[0] = diskConfig

		imgProperties := diskConfig.ImageProperties
		if input.Cdrom != "" {
			cdromStr := input.Cdrom
			image, err := parseIsoInfo(ctx, userCred, cdromStr)
			if err != nil {
				return nil, httperrors.NewInputParameterError("parse cdrom device info error %s", err)
			}
			input.Cdrom = image.Id
			if len(imgProperties) == 0 {
				imgProperties = image.Properties
			}
		}

		if len(imgProperties) == 0 {
			imgProperties = map[string]string{"os_type": "Linux"}
		}

		osType := input.OsType
		osProf, err = osprofile.GetOSProfileFromImageProperties(imgProperties, hypervisor)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Invalid root image: %s", err)
		}

		if len(osProf.Hypervisor) > 0 && len(hypervisor) == 0 {
			hypervisor = osProf.Hypervisor
			input.Hypervisor = hypervisor
		}
		if len(osProf.OSType) > 0 && len(osType) == 0 {
			osType = osProf.OSType
			input.OsType = osType
		}
		input.OsProfile = jsonutils.Marshal(osProf)
	}

	input, err = ValidateScheduleCreateData(ctx, userCred, input, hypervisor)
	if err != nil {
		return nil, err
	}

	hypervisor = input.Hypervisor
	if hypervisor != HYPERVISOR_CONTAINER {
		// support sku here
		var sku *SServerSku
		skuName := input.InstanceType
		if len(skuName) > 0 {
			sku, err := ServerSkuManager.FetchSkuByNameAndHypervisor(skuName, hypervisor, true)
			if err != nil {
				return nil, err
			}

			input.InstanceType = sku.Name
			input.VmemSize = sku.MemorySizeMB
			input.VcpuCount = sku.CpuCoreCount
		} else {
			vmemSize, vcpuCount, err := ValidateMemCpuData(input.VmemSize, input.VcpuCount, input.Hypervisor)
			if err != nil {
				return nil, err
			}

			if vmemSize == 0 {
				return nil, httperrors.NewMissingParameterError("vmem_size")
			}
			if vcpuCount == 0 {
				vcpuCount = 1
			}
			input.VmemSize = vmemSize
			input.VcpuCount = vcpuCount
		}

		dataDiskDefs := []*api.DiskConfig{}
		if sku != nil && sku.AttachedDiskCount > 0 {
			for i := 0; i < sku.AttachedDiskCount; i += 1 {
				dataDisk := &api.DiskConfig{
					SizeMb:  sku.AttachedDiskSizeGB * 1024,
					Backend: strings.ToLower(sku.AttachedDiskType),
				}
				dataDiskDefs = append(dataDiskDefs, dataDisk)
			}
		}

		// start from data disk
		disks := input.Disks
		for idx := 1; idx < len(disks); idx += 1 {
			dataDiskDefs = append(dataDiskDefs, disks[idx])
		}

		rootDiskConfig, err := parseDiskInfo(ctx, userCred, disks[0])
		if err != nil {
			return nil, httperrors.NewGeneralError(err) // should no error
		}
		if len(rootDiskConfig.Backend) == 0 {
			rootDiskConfig.Backend = GetDriver(hypervisor).GetDefaultSysDiskBackend()
		}
		if rootDiskConfig.SizeMb == 0 {
			rootDiskConfig.SizeMb = GetDriver(hypervisor).GetMinimalSysDiskSizeGb() * 1024
		}
		log.Debugf("ROOT DISK: %#v", rootDiskConfig)
		input.Disks[0] = rootDiskConfig
		//data.Set("disk.0", jsonutils.Marshal(rootDiskConfig))

		for i := 0; i < len(dataDiskDefs); i += 1 {
			diskConfig, err := parseDiskInfo(ctx, userCred, dataDiskDefs[i])
			if err != nil {
				return nil, httperrors.NewInputParameterError("parse disk description error %s", err)
			}
			if diskConfig.DiskType == DISK_TYPE_SYS {
				return nil, httperrors.NewBadRequestError("Snapshot error: disk index %d > 0 but disk type is %s", i+1, DISK_TYPE_SYS)
			}
			if len(diskConfig.Backend) == 0 {
				diskConfig.Backend = rootDiskConfig.Backend
			}
			if len(diskConfig.Driver) == 0 {
				diskConfig.Driver = osProf.DiskDriver
			}
			input.Disks[i+1] = diskConfig
		}

		resourceTypeStr := input.ResourceType
		durationStr := input.Duration

		if len(durationStr) > 0 {

			if !userCred.IsAdminAllow(consts.GetServiceType(), manager.KeywordPlural(), "renew") {
				return nil, httperrors.NewForbiddenError("only admin can create prepaid resource")
			}

			if resourceTypeStr == HostResourceTypePrepaidRecycle {
				return nil, httperrors.NewConflictError("cannot create prepaid server on prepaid resource type")
			}

			billingCycle, err := billing.ParseBillingCycle(durationStr)
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid duration %s", durationStr)
			}

			if !GetDriver(hypervisor).IsSupportedBillingCycle(billingCycle) {
				return nil, httperrors.NewInputParameterError("unsupported duration %s", durationStr)
			}

			input.BillingType = BILLING_TYPE_PREPAID
			input.BillingCycle = billingCycle.String()
			// expired_at will be set later by callback
			// data.Add(jsonutils.NewTimeString(billingCycle.EndAt(time.Time{})), "expired_at")

			input.Duration = billingCycle.String()
		}
	}

	netArray := input.Networks
	for idx := 0; idx < len(netArray); idx += 1 {
		netConfig, err := parseNetworkInfo(userCred, netArray[idx])
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
		input.Networks[idx] = netConfig
	}

	isoDevArray := input.IsolatedDevices
	for idx := 0; idx < len(isoDevArray); idx += 1 { // .Contains(fmt.Sprintf("isolated_device.%d", idx)); idx += 1 {
		if input.Backup {
			return nil, httperrors.NewBadRequestError("Cannot create backup with isolated device")
		}
		devConfig, err := IsolatedDeviceManager.parseDeviceInfo(userCred, isoDevArray[idx])
		if err != nil {
			return nil, httperrors.NewInputParameterError("parse isolated device description error %s", err)
		}
		err = IsolatedDeviceManager.isValidDeviceinfo(devConfig)
		if err != nil {
			return nil, err
		}
		input.IsolatedDevices[idx] = devConfig
	}

	keypairId := input.KeypairId
	if len(keypairId) > 0 {
		keypairObj, err := KeypairManager.FetchByIdOrName(userCred, keypairId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Keypair %s not found", keypairId)
		}
		input.KeypairId = keypairObj.GetId()
	} else {
		input.KeypairId = "None" // TODO: ??? None?
	}

	if input.SecgroupId != "" {
		secGrpId := input.SecgroupId
		secGrpObj, err := SecurityGroupManager.FetchByIdOrName(userCred, secGrpId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Secgroup %s not found", secGrpId)
		}
		input.SecgroupId = secGrpObj.GetId()
	} else {
		input.SecgroupId = "default"
	}

	eipStr := input.Eip
	eipBw := input.EipBw
	if len(eipStr) > 0 || eipBw > 0 {
		if !GetDriver(hypervisor).IsSupportEip() {
			return nil, httperrors.NewNotImplementedError("eip not supported for %s", hypervisor)
		}
		if len(eipStr) > 0 {
			eipObj, err := ElasticipManager.FetchByIdOrName(userCred, eipStr)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, httperrors.NewResourceNotFoundError2(ElasticipManager.Keyword(), eipStr)
				} else {
					return nil, httperrors.NewGeneralError(err)
				}
			}

			eip := eipObj.(*SElasticip)
			if eip.Status != EIP_STATUS_READY {
				return nil, httperrors.NewInvalidStatusError("eip %s status invalid %s", eipStr, eip.Status)
			}
			if eip.IsAssociated() {
				return nil, httperrors.NewResourceBusyError("eip %s has been associated", eipStr)
			}
			input.Eip = eipObj.GetId()

			eipRegion := eip.GetRegion()
			preferRegionId, _ := data.GetString("prefer_region_id")
			if len(preferRegionId) > 0 && preferRegionId != eipRegion.Id {
				return nil, httperrors.NewConflictError("cannot assoicate with eip %s: different region", eipStr)
			}
			input.PreferRegion = eipRegion.Id
		} else {
			// create new eip
		}
	}

	/*
		TODO
		group
		for idx := 0; data.Contains(fmt.Sprintf("srvtag.%d", idx)); idx += 1 {

		}*/

	input, err = GetDriver(hypervisor).ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}

	data, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, input.JSON(input))
	if err != nil {
		return nil, err
	}
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}

	if !input.IsSystem {
		err = manager.checkCreateQuota(ctx, userCred, ownerProjId, input,
			input.Backup)
		if err != nil {
			return nil, err
		}
	}

	input.Project = ownerProjId
	return input.JSON(input), nil
}

func (manager *SGuestManager) checkCreateQuota(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, input *api.ServerCreateInput, hasBackup bool) error {
	req := getGuestResourceRequirements(ctx, userCred, input, 1, hasBackup)
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

func getGuestResourceRequirements(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput, count int, hasBackup bool) SQuota {
	vcpuCount := input.VcpuCount
	if vcpuCount == 0 {
		vcpuCount = 1
	}

	vmemSize := input.VmemSize

	diskSize := 0

	for _, diskConfig := range input.Disks {
		diskSize += diskConfig.SizeMb
	}

	devCount := len(input.IsolatedDevices)

	eNicCnt := 0
	iNicCnt := 0
	eBw := 0
	iBw := 0
	for _, netConfig := range input.Networks {
		if isExitNetworkInfo(netConfig) {
			eNicCnt += 1
			eBw += netConfig.BwLimit
		} else {
			iNicCnt += 1
			iBw += netConfig.BwLimit
		}
	}
	if hasBackup {
		vcpuCount = vcpuCount * 2
		vmemSize = vmemSize * 2
		diskSize = diskSize * 2
	}

	eipCnt := 0
	eipBw := input.EipBw
	if eipBw > 0 {
		eipCnt = 1
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
		Eip:            eipCnt * count,
	}
}

func (guest *SGuest) getGuestBackupResourceRequirements(ctx context.Context, userCred mcclient.TokenCredential) SQuota {
	guestDisksSize := guest.getDiskSize()
	return SQuota{
		Cpu:     int(guest.VcpuCount),
		Memory:  guest.VmemSize,
		Storage: guestDisksSize,
	}
}

func (guest *SGuest) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	guest.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	tags := []string{"cpu_bound", "io_bound", "io_hardlimit"}
	appTags := make([]string, 0)
	for _, tag := range tags {
		if data.Contains(tag) {
			appTags = append(appTags, tag)
		}
	}
	guest.setApptags(ctx, appTags, userCred)
	osProfileJson, _ := data.Get("__os_profile__")
	if osProfileJson != nil {
		guest.setOSProfile(ctx, userCred, osProfileJson)
	}

	userData, _ := data.GetString("user_data")
	if len(userData) > 0 {
		guest.setUserData(ctx, userCred, userData)
	}
}

func (guest *SGuest) setApptags(ctx context.Context, appTags []string, userCred mcclient.TokenCredential) {
	err := guest.SetMetadata(ctx, "app_tags", strings.Join(appTags, ","), userCred)
	if err != nil {
		log.Errorln(err)
	}
}

func (manager *SGuestManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := new(api.ServerCreateInput)
	data.Unmarshal(input)
	pendingUsage := getGuestResourceRequirements(ctx, userCred, input, len(items), input.Backup)
	RunBatchCreateTask(ctx, items, userCred, data, pendingUsage, "GuestBatchCreateTask")
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
	networks, err := self.GetNetworks("")
	if err != nil {
		return bw
	}
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

	if db.IsAdminAllowGet(userCred, self) {
		host := self.GetHost()
		if host != nil {
			extra.Add(jsonutils.NewString(host.Name), "host")
		}
	}
	extra.Add(jsonutils.NewString(strings.Join(self.getRealIPs(), ",")), "ips")
	eip, _ := self.GetEip()
	if eip != nil {
		extra.Add(jsonutils.NewString(eip.IpAddr), "eip")
		extra.Add(jsonutils.NewString(eip.Mode), "eip_mode")
	}
	extra.Add(jsonutils.NewInt(int64(self.getDiskSize())), "disk")
	// flavor??
	// extra.Add(jsonutils.NewString(self.getFlavorName()), "flavor")
	extra.Add(jsonutils.NewString(self.getKeypairName()), "keypair")
	extra.Add(jsonutils.NewInt(int64(self.getExtBandwidth())), "ext_bw")

	extra.Add(jsonutils.NewString(self.GetSecgroupName()), "secgroup")

	if secgroups := self.getSecgroupJson(); len(secgroups) > 0 {
		extra.Add(jsonutils.NewArray(secgroups...), "secgroups")
	}

	if self.PendingDeleted {
		pendingDeletedAt := self.PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
		extra.Add(jsonutils.NewString(timeutils.FullIsoTime(pendingDeletedAt)), "auto_delete_at")
	}

	isGpu := jsonutils.JSONFalse
	if self.isGpu() {
		isGpu = jsonutils.JSONTrue
	}
	extra.Add(isGpu, "is_gpu")

	extra.Add(jsonutils.JSONNull, "cdrom")
	if cdrom := self.getCdrom(); cdrom != nil {
		extra.Set("cdrom", jsonutils.NewString(cdrom.GetDetails()))
	}

	return self.moreExtraInfo(extra)
}

func (self *SGuest) moreExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	/*zone := self.getZone()
	if zone != nil {
		extra.Add(jsonutils.NewString(zone.GetId()), "zone_id")
		extra.Add(jsonutils.NewString(zone.GetName()), "zone")
		if len(zone.ExternalId) > 0 {
			extra.Add(jsonutils.NewString(zone.ExternalId), "zone_external_id")
		}

		region := zone.GetRegion()
		if region != nil {
			extra.Add(jsonutils.NewString(region.Id), "region_id")
			extra.Add(jsonutils.NewString(region.Name), "region")

			if len(region.ExternalId) > 0 {
				extra.Add(jsonutils.NewString(region.ExternalId), "region_external_id")
			}
		}

		host := self.GetHost()
		if host != nil {
			provider := host.GetCloudprovider()
			if provider != nil {
				extra.Add(jsonutils.NewString(host.ManagerId), "manager_id")
				extra.Add(jsonutils.NewString(provider.GetName()), "manager")
			}
		}
	}*/

	extra.Add(self.getDisksInfoDetails(), "disks_info")
	extra.Add(jsonutils.NewString(self.getIsolatedDeviceDetails()), "isolated_devices")

	if len(self.BackupHostId) > 0 {
		backupHost := HostManager.FetchHostById(self.BackupHostId)
		extra.Set("backup_host_name", jsonutils.NewString(backupHost.Name))
		extra.Set("backup_host_status", jsonutils.NewString(backupHost.HostStatus))
	}

	host := self.GetHost()
	if host != nil {
		info := host.getCloudProviderInfo()
		extra.Update(jsonutils.Marshal(&info))
	}

	err := self.CanPerformPrepaidRecycle()
	if err != nil {
		extra.Add(jsonutils.JSONFalse, "can_recycle")
	} else {
		extra.Add(jsonutils.JSONTrue, "can_recycle")
	}

	guestnetworks, _ := self.GetNetworks("")
	if len(guestnetworks) > 0 {
		guestnetwork := guestnetworks[0]
		network := guestnetwork.GetNetwork()
		if network != nil {
			vpc := network.GetVpc()
			extra.Set("vpc_id", jsonutils.NewString(vpc.Id))
			extra.Set("vpc", jsonutils.NewString(vpc.Name))
		}
	}

	return extra
}

func (self *SGuest) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}

	extra.Add(jsonutils.NewString(self.getNetworksDetails()), "networks")
	extra.Add(jsonutils.NewString(self.getDisksDetails()), "disks")
	extra.Add(jsonutils.NewInt(int64(self.getDiskSize())), "disk")
	cdrom := self.getCdrom()
	if cdrom != nil {
		extra.Add(jsonutils.NewString(cdrom.GetDetails()), "cdrom")
	}
	// extra.Add(jsonutils.NewString(self.getFlavorName()), "flavor")
	extra.Add(jsonutils.NewString(self.getKeypairName()), "keypair")
	extra.Add(jsonutils.NewString(self.GetSecgroupName()), "secgroup")

	if secgroups := self.getSecgroupJson(); len(secgroups) > 0 {
		extra.Add(jsonutils.NewArray(secgroups...), "secgroups")
	}

	extra.Add(jsonutils.NewString(strings.Join(self.getIPs(), ",")), "ips")
	extra.Add(jsonutils.NewString(self.getSecurityGroupsRules()), "security_rules")
	osName := self.GetOS()
	if len(osName) > 0 {
		extra.Add(jsonutils.NewString(osName), "os_name")
		if len(self.OsType) == 0 {
			extra.Add(jsonutils.NewString(osName), "os_type")
		}
	}
	if metaData, err := self.GetAllMetadata(userCred); err == nil {
		extra.Add(jsonutils.Marshal(metaData), "metadata")
	}
	if db.IsAdminAllowGet(userCred, self) {
		host := self.GetHost()
		if host != nil {
			extra.Add(jsonutils.NewString(host.GetName()), "host")
		}
		extra.Add(jsonutils.NewString(self.getAdminSecurityRules()), "admin_security_rules")
	}
	eip, _ := self.GetEip()
	if eip != nil {
		extra.Add(jsonutils.NewString(eip.IpAddr), "eip")
		extra.Add(jsonutils.NewString(eip.Mode), "eip_mode")
	}

	isGpu := jsonutils.JSONFalse
	if self.isGpu() {
		isGpu = jsonutils.JSONTrue
	}
	extra.Add(isGpu, "is_gpu")

	if self.IsPrepaidRecycle() {
		extra.Add(jsonutils.JSONTrue, "is_prepaid_recycle")
	} else {
		extra.Add(jsonutils.JSONFalse, "is_prepaid_recycle")
	}

	return self.moreExtraInfo(extra), nil
}

func (manager *SGuestManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	exportKeys, _ := query.GetString("export_keys")
	keys := strings.Split(exportKeys, ",")

	// guest_id as filter key
	if utils.IsInStringArray("ips", keys) {
		guestIpsQuery := GuestnetworkManager.Query("guest_id").GroupBy("guest_id")
		guestIpsQuery.AppendField(sqlchemy.GROUP_CONCAT("concat_ip_addr", guestIpsQuery.Field("ip_addr")))
		ipsSubQuery := guestIpsQuery.SubQuery()
		guestIpsQuery.DebugQuery()
		q.LeftJoin(ipsSubQuery, sqlchemy.Equals(q.Field("id"), ipsSubQuery.Field("guest_id")))
		q.AppendField(ipsSubQuery.Field("concat_ip_addr"))
	}
	if utils.IsInStringArray("disk", keys) {
		guestDisksQuery := GuestdiskManager.Query("guest_id", "disk_id").GroupBy("guest_id")
		diskQuery := DiskManager.Query("id", "disk_size").SubQuery()
		guestDisksQuery.Join(diskQuery, sqlchemy.Equals(diskQuery.Field("id"), guestDisksQuery.Field("disk_id")))
		guestDisksQuery.AppendField(sqlchemy.SUM("disk_size", diskQuery.Field("disk_size")))
		guestDisksSubQuery := guestDisksQuery.SubQuery()
		guestDisksSubQuery.DebugQuery()
		q.LeftJoin(guestDisksSubQuery, sqlchemy.Equals(q.Field("id"), guestDisksSubQuery.
			Field("guest_id")))
		q.AppendField(guestDisksSubQuery.Field("disk_size"))
	}
	if utils.IsInStringArray("eip", keys) {
		eipsQuery := ElasticipManager.Query("associate_id", "ip_addr").Equals("associate_type", "server").GroupBy("associate_id")
		eipsSubQuery := eipsQuery.SubQuery()
		eipsSubQuery.DebugQuery()
		q.LeftJoin(eipsSubQuery, sqlchemy.Equals(q.Field("id"), eipsSubQuery.Field("associate_id")))
		q.AppendField(eipsSubQuery.Field("ip_addr", "eip"))
	}

	// host_id as filter key
	if utils.IsInStringArray("region", keys) {
		zoneQuery := ZoneManager.Query("id", "cloudregion_id").SubQuery()
		hostQuery := HostManager.Query("id", "zone_id").GroupBy("id")
		cloudregionQuery := CloudregionManager.Query("id", "name").SubQuery()
		hostQuery.LeftJoin(zoneQuery, sqlchemy.Equals(hostQuery.Field("zone_id"), zoneQuery.Field("id"))).
			LeftJoin(cloudregionQuery, sqlchemy.OR(sqlchemy.Equals(cloudregionQuery.Field("id"),
				zoneQuery.Field("cloudregion_id")), sqlchemy.Equals(cloudregionQuery.Field("id"), "default")))
		hostQuery.AppendField(cloudregionQuery.Field("name", "region"))
		hostSubQuery := hostQuery.SubQuery()
		q.LeftJoin(hostSubQuery, sqlchemy.Equals(q.Field("host_id"), hostSubQuery.Field("id")))
		q.AppendField(hostSubQuery.Field("region"))
	}
	if utils.IsInStringArray("manager", keys) {
		hostQuery := HostManager.Query("id", "manager_id").GroupBy("id")
		cloudProviderQuery := CloudproviderManager.Query("id", "name").SubQuery()
		hostQuery.LeftJoin(cloudProviderQuery, sqlchemy.Equals(hostQuery.Field("manager_id"),
			cloudProviderQuery.Field("id")))
		hostQuery.AppendField(cloudProviderQuery.Field("name", "manager"))
		hostSubQuery := hostQuery.SubQuery()
		q.LeftJoin(hostSubQuery, sqlchemy.Equals(q.Field("host_id"), hostSubQuery.Field("id")))
		q.AppendField(hostSubQuery.Field("manager"))
	}
	return q, nil
}

func (manager *SGuestManager) GetExportExtraKeys(ctx context.Context, query jsonutils.JSONObject, rowMap map[string]string) *jsonutils.JSONDict {
	res := manager.SStatusStandaloneResourceBaseManager.GetExportExtraKeys(ctx, query, rowMap)
	exportKeys, _ := query.GetString("export_keys")
	keys := strings.Split(exportKeys, ",")
	if ips, ok := rowMap["concat_ip_addr"]; ok && len(ips) > 0 {
		res.Set("ips", jsonutils.NewString(ips))
	}
	if eip, ok := rowMap["eip"]; ok && len(eip) > 0 {
		res.Set("eip", jsonutils.NewString(eip))
	}
	if disk, ok := rowMap["disk_size"]; ok {
		res.Set("disk", jsonutils.NewString(disk))
	}
	if region, ok := rowMap["region"]; ok && len(region) > 0 {
		res.Set("region", jsonutils.NewString(region))
	}
	if manager, ok := rowMap["manager"]; ok && len(manager) > 0 {
		res.Set("manager", jsonutils.NewString(manager))
	}
	if utils.IsInStringArray("tenant", keys) {
		if projectId, ok := rowMap["tenant_id"]; ok {
			tenant, err := db.TenantCacheManager.FetchTenantById(ctx, projectId)
			if err == nil {
				res.Set("tenant", jsonutils.NewString(tenant.GetName()))
			}
		}
	}
	if utils.IsInStringArray("os_distribution", keys) {
		if osType, ok := rowMap["os_type"]; ok {
			res.Set("os_distribution", jsonutils.NewString(osType))
		}
	}
	return res
}

func (self *SGuest) getNetworksDetails() string {
	guestnets, err := self.GetNetworks("")
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	for _, nic := range guestnets {
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

func (self *SGuest) getDisksInfoDetails() *jsonutils.JSONArray {
	details := jsonutils.NewArray()
	for _, disk := range self.GetDisks() {
		details.Add(disk.GetDetailedJson())
	}
	return details
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

func (self *SGuest) getNotifyIps() string {
	ips := self.getRealIPs()
	vips := self.getVirtualIPs()
	if vips != nil {
		ips = append(ips, vips...)
	}
	return strings.Join(ips, ",")
}

func (self *SGuest) getRealIPs() []string {
	guestnets, err := self.GetNetworks("")
	if err != nil {
		return nil
	}
	ips := make([]string, 0)
	for _, nic := range guestnets {
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
		groupnets, err := group.GetNetworks()
		if err != nil {
			continue
		}
		for _, groupnetwork := range groupnets {
			ips = append(ips, groupnetwork.IpAddr)
		}
	}
	return ips
}

func (self *SGuest) getIPs() []string {
	ips := self.getRealIPs()
	vips := self.getVirtualIPs()
	ips = append(ips, vips...)
	/*eip, _ := self.GetEip()
	if eip != nil {
		ips = append(ips, eip.IpAddr)
	}*/
	return ips
}

func (self *SGuest) getZone() *SZone {
	host := self.GetHost()
	if host != nil {
		return host.GetZone()
	}
	return nil
}

func (self *SGuest) getRegion() *SCloudregion {
	zone := self.getZone()
	if zone != nil {
		return zone.GetRegion()
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

func (self *SGuest) getSecgroupJson() []jsonutils.JSONObject {
	secgroups := []jsonutils.JSONObject{}
	for _, secGrp := range self.GetSecgroups() {
		secgroups = append(secgroups, secGrp.getDesc())
	}
	return secgroups
}

func (self *SGuest) GetSecgroups() []SSecurityGroup {
	secgrpQuery := SecurityGroupManager.Query()
	secgrpQuery.Filter(
		sqlchemy.OR(
			sqlchemy.Equals(secgrpQuery.Field("id"), self.SecgrpId),
			sqlchemy.In(secgrpQuery.Field("id"), GuestsecgroupManager.Query("secgroup_id").Equals("guest_id", self.Id).SubQuery()),
		),
	)
	secgroups := []SSecurityGroup{}
	if err := db.FetchModelObjects(SecurityGroupManager, secgrpQuery, &secgroups); err != nil {
		log.Errorf("Get security group error: %v", err)
		return nil
	}
	return secgroups
}

func (self *SGuest) getSecgroup() *SSecurityGroup {
	return SecurityGroupManager.FetchSecgroupById(self.SecgrpId)
}

func (self *SGuest) getAdminSecgroup() *SSecurityGroup {
	return SecurityGroupManager.FetchSecgroupById(self.AdminSecgrpId)
}

func (self *SGuest) GetSecgroupName() string {
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

func (self *SGuest) GetSecRules() []secrules.SecurityRule {
	return self.getSecRules()
}

func (self *SGuest) getSecRules() []secrules.SecurityRule {
	if secgrp := self.getSecgroup(); secgrp != nil {
		return secgrp.GetSecRules("")
	}
	if rule, err := secrules.ParseSecurityRule(options.Options.DefaultSecurityRules); err == nil {
		return []secrules.SecurityRule{*rule}
	} else {
		log.Errorf("Default SecurityRules error: %v", err)
	}
	return []secrules.SecurityRule{}
}

func (self *SGuest) getSecurityRules() string {
	secgrp := self.getSecgroup()
	if secgrp != nil {
		return secgrp.getSecurityRuleString("")
	} else {
		return options.Options.DefaultSecurityRules
	}
}

//获取多个安全组规则，优先级降序排序
func (self *SGuest) getSecurityGroupsRules() string {
	secgroups := self.GetSecgroups()
	secgroupids := []string{}
	for _, secgroup := range secgroups {
		secgroupids = append(secgroupids, secgroup.Id)
	}
	q := SecurityGroupRuleManager.Query()
	q.Filter(sqlchemy.In(q.Field("secgroup_id"), secgroupids)).Desc(q.Field("priority"))
	secrules := []SSecurityGroupRule{}
	if err := db.FetchModelObjects(SecurityGroupRuleManager, q, &secrules); err != nil {
		log.Errorf("Get rules error: %v", err)
		return options.Options.DefaultSecurityRules
	}
	rules := []string{}
	for _, rule := range secrules {
		rules = append(rules, rule.String())
	}
	return strings.Join(rules, SECURITY_GROUP_SEPARATOR)
}

func (self *SGuest) getAdminSecurityRules() string {
	secgrp := self.getAdminSecgroup()
	if secgrp != nil {
		return secgrp.getSecurityRuleString("")
	} else {
		return options.Options.DefaultAdminSecurityRules
	}
}

func (self *SGuest) isGpu() bool {
	return len(self.GetIsolatedDevices()) != 0
}

func (self *SGuest) GetIsolatedDevices() []SIsolatedDevice {
	return IsolatedDeviceManager.findAttachedDevicesOfGuest(self)
}

func (self *SGuest) syncRemoveCloudVM(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	if self.BillingType == BILLING_TYPE_PREPAID {
		diff, err := db.Update(self, func() error {
			self.BillingType = BILLING_TYPE_POSTPAID
			self.ExpiredAt = time.Time{}
			return nil
		})
		if err != nil {
			return err
		}
		db.OpsLog.LogSyncUpdate(self, diff, userCred)
	}

	return self.SetStatus(userCred, VM_UNKNOWN, "Sync lost")
}

func (self *SGuest) syncWithCloudVM(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, host *SHost, extVM cloudprovider.ICloudVM, projectId string) error {
	recycle := false

	if provider.GetFactory().IsSupportPrepaidResources() && self.IsPrepaidRecycle() {
		recycle = true
	}

	// metaData := extVM.GetMetadata()
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		extVM.Refresh()
		// self.Name = extVM.GetName()
		self.Status = extVM.GetStatus()
		self.VcpuCount = extVM.GetVcpuCount()
		self.BootOrder = extVM.GetBootOrder()
		self.Vga = extVM.GetVga()
		self.Vdi = extVM.GetVdi()
		self.OsType = extVM.GetOSType()
		self.Bios = extVM.GetBios()
		self.Machine = extVM.GetMachine()
		if !recycle {
			self.HostId = host.Id
		}

		instanceType := extVM.GetInstanceType()

		if len(instanceType) > 0 {
			self.InstanceType = instanceType
		}

		if extVM.GetHypervisor() == HYPERVISOR_AWS {
			sku, err := ServerSkuManager.FetchSkuByNameAndHypervisor(instanceType, extVM.GetHypervisor(), false)
			if err == nil {
				self.VmemSize = sku.MemorySizeMB
			} else {
				self.VmemSize = extVM.GetVmemSizeMB()
			}
		} else {
			self.VmemSize = extVM.GetVmemSizeMB()
		}

		self.Hypervisor = extVM.GetHypervisor()

		self.IsEmulated = extVM.IsEmulated()

		if provider.GetFactory().IsSupportPrepaidResources() && !recycle {
			self.BillingType = extVM.GetBillingType()
			self.ExpiredAt = extVM.GetExpiredAt()
		}

		return nil
	})
	if err != nil {
		log.Errorf("%s", err)
		return err
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	SyncCloudProject(userCred, self, projectId, extVM, host.ManagerId)

	if provider.GetFactory().IsSupportPrepaidResources() && recycle {
		vhost := self.GetHost()
		err = vhost.syncWithCloudPrepaidVM(extVM, host)
		if err != nil {
			return err
		}
	}

	return nil
}

func (manager *SGuestManager) newCloudVM(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, host *SHost, extVM cloudprovider.ICloudVM, projectId string) (*SGuest, error) {

	guest := SGuest{}
	guest.SetModelManager(manager)

	guest.Status = extVM.GetStatus()
	guest.ExternalId = extVM.GetGlobalId()
	guest.Name = db.GenerateName(manager, projectId, extVM.GetName())
	guest.VcpuCount = extVM.GetVcpuCount()
	guest.BootOrder = extVM.GetBootOrder()
	guest.Vga = extVM.GetVga()
	guest.Vdi = extVM.GetVdi()
	guest.OsType = extVM.GetOSType()
	guest.Bios = extVM.GetBios()
	guest.Machine = extVM.GetMachine()
	guest.Hypervisor = extVM.GetHypervisor()

	guest.IsEmulated = extVM.IsEmulated()

	if provider.GetFactory().IsSupportPrepaidResources() {
		guest.BillingType = extVM.GetBillingType()
		guest.ExpiredAt = extVM.GetExpiredAt()
	}

	guest.HostId = host.Id

	instanceType := extVM.GetInstanceType()

	/*zoneExtId, err := metaData.GetString("zone_ext_id")
	if err != nil {
		log.Errorf("get zone external id fail %s", err)
	}

	isku, err := ServerSkuManager.FetchByZoneExtId(zoneExtId, instanceType)
	if err != nil {
		log.Errorf("get sku zone %s instance type %s fail %s", zoneExtId, instanceType, err)
	} else {
		guest.SkuId = isku.GetId()
	}*/

	if len(instanceType) > 0 {
		guest.InstanceType = instanceType
	}

	if extVM.GetHypervisor() == HYPERVISOR_AWS {
		sku, err := ServerSkuManager.FetchSkuByNameAndHypervisor(instanceType, extVM.GetHypervisor(), false)
		if err == nil {
			guest.VmemSize = sku.MemorySizeMB
		} else {
			guest.VmemSize = extVM.GetVmemSizeMB()
		}
	} else {
		guest.VmemSize = extVM.GetVmemSizeMB()
	}

	err := manager.TableSpec().Insert(&guest)
	if err != nil {
		log.Errorf("Insert fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &guest, projectId, extVM, host.ManagerId)

	db.OpsLog.LogEvent(&guest, db.ACT_CREATE, guest.GetShortDesc(ctx), userCred)

	if guest.Status == VM_RUNNING {
		db.OpsLog.LogEvent(&guest, db.ACT_START, guest.GetShortDesc(ctx), userCred)
	}

	return &guest, nil
}

func (manager *SGuestManager) TotalCount(
	projectId string, rangeObj db.IStandaloneModel,
	status []string, hypervisors []string,
	includeSystem bool, pendingDelete bool,
	hostTypes []string, resourceTypes []string, providers []string,
) SGuestCountStat {
	return totalGuestResourceCount(projectId, rangeObj, status, hypervisors, includeSystem, pendingDelete, hostTypes, resourceTypes, providers)
}

func (self *SGuest) detachNetworks(ctx context.Context, userCred mcclient.TokenCredential, gns []SGuestnetwork, reserve bool, deploy bool) error {
	err := GuestnetworkManager.DeleteGuestNics(ctx, userCred, gns, reserve)
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

func (self *SGuest) getAttach2NetworkCount(net *SNetwork) int {
	q := GuestnetworkManager.Query()
	q = q.Equals("guest_id", self.Id).Equals("network_id", net.Id)
	return q.Count()
}

func (self *SGuest) getMaxNicIndex() int8 {
	nics, err := self.GetNetworks("")
	if err != nil {
		return -1
	}
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

func (self *SGuest) Attach2Network(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork,
	pendingUsage quotas.IQuota,
	address string,
	driver string, bwLimit int, virtual bool,
	reserved bool, allocDir IPAddlocationDirection, requireDesignatedIP bool,
	nicConfs []SNicConfig) ([]SGuestnetwork, error) {

	firstNic, err := self.attach2NetworkOnce(ctx, userCred, network, pendingUsage, address, driver, bwLimit, virtual,
		reserved, allocDir, requireDesignatedIP, nicConfs[0], "")
	if err != nil {
		return nil, err
	}
	retNics := []SGuestnetwork{*firstNic}
	if len(nicConfs) > 1 {
		firstMac, _ := netutils2.ParseMac(firstNic.MacAddr)
		for i := 1; i < len(nicConfs); i += 1 {
			if len(nicConfs[i].Mac) == 0 {
				nicConfs[i].Mac = firstMac.Add(i).String()
			}
			gn, err := self.attach2NetworkOnce(ctx, userCred, network, pendingUsage, "", firstNic.Driver, 0, true,
				false, allocDir, false, nicConfs[i], firstNic.MacAddr)
			if err != nil {
				return retNics, err
			}
			retNics = append(retNics, *gn)
		}
	}
	return retNics, nil
}

func (self *SGuest) attach2NetworkOnce(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork,
	pendingUsage quotas.IQuota,
	address string,
	driver string, bwLimit int, virtual bool,
	reserved bool, allocDir IPAddlocationDirection, requireDesignatedIP bool,
	nicConf SNicConfig, teamWithMac string) (*SGuestnetwork, error) {
	/*
		allow a guest attach to a network 2 times
	*/
	/*if self.getAttach2NetworkCount(network) > MAX_GUESTNIC_TO_SAME_NETWORK {
		return nil, fmt.Errorf("Guest has been attached to network %s", network.Name)
	}*/
	if nicConf.Index < 0 {
		nicConf.Index = self.getMaxNicIndex()
	}
	if len(driver) == 0 {
		osProf := self.getOSProfile()
		driver = osProf.NetDriver
	}
	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)

	guestnic, err := GuestnetworkManager.newGuestNetwork(ctx, userCred, self, network,
		nicConf.Index, address, nicConf.Mac, driver, bwLimit, virtual, reserved,
		allocDir, requireDesignatedIP, nicConf.Ifname, teamWithMac)
	if err != nil {
		return nil, err
	}
	network.updateDnsRecord(guestnic, true)
	network.updateGuestNetmap(guestnic)
	bwLimit = guestnic.getBandwidth()
	if pendingUsage != nil && len(teamWithMac) == 0 {
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
			log.Warningf("QuotaManager.CancelPendingUsage fail %s", err)
		}
	}
	if len(address) == 0 {
		address = guestnic.IpAddr
	}
	db.OpsLog.LogAttachEvent(ctx, self, network, userCred, guestnic.GetShortDesc(ctx))
	return guestnic, nil
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
			return host.getNetworkOfIPOnHost(ip)
		}
	}
	localNetObj, err := NetworkManager.FetchByExternalId(vnet.GetGlobalId())
	if err != nil {
		return nil, fmt.Errorf("Cannot find network of external_id %s: %v", vnet.GetGlobalId(), err)
	}
	localNet := localNetObj.(*SNetwork)
	return localNet, nil
}

func (self *SGuest) SyncVMNics(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, vnics []cloudprovider.ICloudNic) compare.SyncResult {
	result := compare.SyncResult{}

	guestnics, err := self.GetNetworks("")
	if err != nil {
		result.Error(err)
		return result
	}

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
		err := self.detachNetworks(ctx, userCred, []SGuestnetwork{*remove.nic}, remove.reserve, false)
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
		// check if the IP has been occupied, if yes, release the IP
		gn, err := GuestnetworkManager.getGuestNicByIP(add.nic.GetIP(), add.net.Id)
		if err != nil {
			result.AddError(err)
			continue
		}
		if gn != nil {
			err = gn.Detach(ctx, userCred)
			if err != nil {
				result.AddError(err)
				continue
			}
		}
		nicConf := SNicConfig{
			Mac:    add.nic.GetMAC(),
			Index:  -1,
			Ifname: "",
		}
		_, err = self.Attach2Network(ctx, userCred, add.net, nil, add.nic.GetIP(),
			add.nic.GetDriver(), 0, false, add.reserve, IPAllocationDefault, true, []SNicConfig{nicConf})
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

func (self *SGuest) AttachDisk(ctx context.Context, disk *SDisk, userCred mcclient.TokenCredential, driver string, cache string, mountpoint string) error {
	return self.attach2Disk(ctx, disk, userCred, driver, cache, mountpoint)
}

func (self *SGuest) attach2Disk(ctx context.Context, disk *SDisk, userCred mcclient.TokenCredential, driver string, cache string, mountpoint string) error {
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
		db.OpsLog.LogAttachEvent(ctx, self, disk, userCred, nil)
	}
	return err
}

type sSyncDiskPair struct {
	disk  *SDisk
	vdisk cloudprovider.ICloudDisk
}

func (self *SGuest) SyncVMDisks(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, host *SHost, vdisks []cloudprovider.ICloudDisk, projectId string) compare.SyncResult {
	result := compare.SyncResult{}

	newdisks := make([]sSyncDiskPair, 0)
	for i := 0; i < len(vdisks); i += 1 {
		if len(vdisks[i].GetGlobalId()) == 0 {
			continue
		}
		disk, err := DiskManager.syncCloudDisk(ctx, userCred, provider, vdisks[i], i, projectId)
		if err != nil {
			log.Errorf("syncCloudDisk error: %v", err)
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
		err := self.attach2Disk(ctx, needAdds[i].disk, userCred, vdisk.GetDriver(), vdisk.GetCacheMode(), vdisk.GetMountpoint())
		if err != nil {
			log.Errorf("attach2Disk error: %v", err)
			result.AddError(err)
		} else {
			result.Add()
		}
	}
	return result
}

func filterGuestByRange(q *sqlchemy.SQuery, rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string) *sqlchemy.SQuery {
	hosts := HostManager.Query().SubQuery()

	q = q.Join(hosts, sqlchemy.Equals(hosts.Field("id"), q.Field("host_id")))
	q = q.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
	// q = q.Filter(sqlchemy.Equals(hosts.Field("host_status"), HOST_ONLINE))

	q = AttachUsageQuery(q, hosts, hostTypes, resourceTypes, providers, rangeObj)
	return q
}

type SGuestCountStat struct {
	TotalGuestCount       int
	TotalCpuCount         int
	TotalMemSize          int
	TotalDiskSize         int
	TotalIsolatedCount    int
	TotalBackupGuestCount int
	TotalBackupCpuCount   int
	TotalBackupMemSize    int
	TotalBackupDiskSize   int
}

func totalGuestResourceCount(
	projectId string,
	rangeObj db.IStandaloneModel,
	status []string,
	hypervisors []string,
	includeSystem bool,
	pendingDelete bool,
	hostTypes []string,
	resourceTypes []string,
	providers []string,
) SGuestCountStat {

	guestdisks := GuestdiskManager.Query().SubQuery()
	disks := DiskManager.Query().SubQuery()

	diskQuery := guestdisks.Query(guestdisks.Field("guest_id"), sqlchemy.SUM("guest_disk_size", disks.Field("disk_size")))
	diskQuery = diskQuery.Join(disks, sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id")))
	diskQuery = diskQuery.GroupBy(guestdisks.Field("guest_id"))
	diskSubQuery := diskQuery.SubQuery()

	backupDiskQuery := guestdisks.Query(guestdisks.Field("guest_id"), sqlchemy.SUM("guest_disk_size", disks.Field("disk_size")))
	backupDiskQuery = backupDiskQuery.LeftJoin(disks, sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id")))
	backupDiskQuery = backupDiskQuery.Filter(sqlchemy.IsNotEmpty(disks.Field("backup_storage_id")))
	backupDiskQuery = backupDiskQuery.GroupBy(guestdisks.Field("guest_id"))

	diskBackupSubQuery := backupDiskQuery.SubQuery()
	// diskBackupSubQuery := diskQuery.IsNotEmpty("backup_storage_id").SubQuery()

	isolated := IsolatedDeviceManager.Query().SubQuery()

	isoDevQuery := isolated.Query(isolated.Field("guest_id"), sqlchemy.COUNT("device_sum"))
	isoDevQuery = isoDevQuery.Filter(sqlchemy.IsNotNull(isolated.Field("guest_id")))
	isoDevQuery = isoDevQuery.GroupBy(isolated.Field("guest_id"))

	isoDevSubQuery := isoDevQuery.SubQuery()

	guests := GuestManager.Query().SubQuery()
	guestBackupSubQuery := GuestManager.Query(
		"id",
		"vcpu_count",
		"vmem_size",
	).IsNotEmpty("backup_host_id").SubQuery()

	q := guests.Query(sqlchemy.COUNT("total_guest_count"),
		sqlchemy.SUM("total_cpu_count", guests.Field("vcpu_count")),
		sqlchemy.SUM("total_mem_size", guests.Field("vmem_size")),
		sqlchemy.SUM("total_disk_size", diskSubQuery.Field("guest_disk_size")),
		sqlchemy.SUM("total_isolated_count", isoDevSubQuery.Field("device_sum")),
		sqlchemy.SUM("total_backup_disk_size", diskBackupSubQuery.Field("guest_disk_size")),
		sqlchemy.SUM("total_backup_cpu_count", guestBackupSubQuery.Field("vcpu_count")),
		sqlchemy.SUM("total_backup_mem_size", guestBackupSubQuery.Field("vmem_size")),
		sqlchemy.COUNT("total_backup_guest_count", guestBackupSubQuery.Field("id")),
	)

	q = q.LeftJoin(guestBackupSubQuery, sqlchemy.Equals(guestBackupSubQuery.Field("id"), guests.Field("id")))

	q = q.LeftJoin(diskSubQuery, sqlchemy.Equals(diskSubQuery.Field("guest_id"), guests.Field("id")))
	q = q.LeftJoin(diskBackupSubQuery, sqlchemy.Equals(diskBackupSubQuery.Field("guest_id"), guests.Field("id")))

	q = q.LeftJoin(isoDevSubQuery, sqlchemy.Equals(isoDevSubQuery.Field("guest_id"), guests.Field("id")))

	q = filterGuestByRange(q, rangeObj, hostTypes, resourceTypes, providers)

	if len(projectId) > 0 {
		q = q.Filter(sqlchemy.Equals(guests.Field("tenant_id"), projectId))
	}
	if len(status) > 0 {
		q = q.Filter(sqlchemy.In(guests.Field("status"), status))
	}
	if len(hypervisors) > 0 {
		q = q.Filter(sqlchemy.In(guests.Field("hypervisor"), hypervisors))
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
	stat.TotalCpuCount += stat.TotalBackupCpuCount
	stat.TotalMemSize += stat.TotalBackupMemSize
	stat.TotalDiskSize += stat.TotalBackupDiskSize
	return stat
}

func (self *SGuest) getDefaultNetworkConfig() *api.NetworkConfig {
	netConf := api.NetworkConfig{}
	netConf.BwLimit = options.Options.DefaultBandwidth
	osProf := self.getOSProfile()
	netConf.Driver = osProf.NetDriver
	return &netConf
}

func (self *SGuest) CreateNetworksOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netArray []*api.NetworkConfig, pendingUsage quotas.IQuota) error {
	if len(netArray) == 0 {
		netConfig := self.getDefaultNetworkConfig()
		_, err := self.attach2RandomNetwork(ctx, userCred, host, netConfig, pendingUsage)
		return err
	}
	for _, netConfig := range netArray {
		netConfig, err := parseNetworkInfo(userCred, netConfig)
		if err != nil {
			return err
		}
		_, err = self.attach2NetworkDesc(ctx, userCred, host, netConfig, pendingUsage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) attach2NetworkDesc(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]SGuestnetwork, error) {
	var gns []SGuestnetwork
	var err1, err2 error
	if len(netConfig.Network) > 0 {
		gns, err1 = self.attach2NamedNetworkDesc(ctx, userCred, host, netConfig, pendingUsage)
		if err1 == nil {
			return gns, nil
		}
	}
	gns, err2 = self.attach2RandomNetwork(ctx, userCred, host, netConfig, pendingUsage)
	if err2 == nil {
		return gns, nil
	}
	if err1 != nil {
		return nil, fmt.Errorf("%s/%s", err1, err2)
	} else {
		return nil, err2
	}
}

func (self *SGuest) attach2NamedNetworkDesc(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]SGuestnetwork, error) {
	driver := self.GetDriver()
	net, nicConfs, allocDir := driver.GetNamedNetworkConfiguration(self, userCred, host, netConfig)
	if net != nil {
		if len(nicConfs) == 0 {
			return nil, fmt.Errorf("no avaialble network interface?")
		}
		gn, err := self.Attach2Network(ctx, userCred, net, pendingUsage, netConfig.Address, netConfig.Driver, netConfig.BwLimit, netConfig.Vip, netConfig.Reserved, allocDir, false, nicConfs)
		if err != nil {
			log.Errorf("Attach2Network fail %s", err)
			return nil, err
		} else {
			return gn, nil
		}
	} else {
		return nil, fmt.Errorf("Network %s not available", netConfig.Network)
	}
}

func (self *SGuest) attach2RandomNetwork(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]SGuestnetwork, error) {
	driver := self.GetDriver()
	return driver.Attach2RandomNetwork(self, ctx, userCred, host, netConfig, pendingUsage)
}

func (self *SGuest) CreateDisksOnHost(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	host *SHost,
	disks []*api.DiskConfig,
	pendingUsage quotas.IQuota,
	inheritBilling bool,
	isWithServerCreate bool,
	candidate *schedapi.CandidateResource,
) error {
	for idx := 0; idx < len(disks); idx += 1 {
		diskConfig, err := parseDiskInfo(ctx, userCred, disks[idx])
		if err != nil {
			return err
		}
		disk, err := self.createDiskOnHost(ctx, userCred, host, diskConfig, pendingUsage, inheritBilling, isWithServerCreate, candidate.Disks[idx])
		if err != nil {
			return err
		}
		diskConfig.DiskId = disk.Id
		disks[idx] = diskConfig
	}
	return nil
}

func (self *SGuest) createDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage,
	diskConfig *api.DiskConfig, pendingUsage quotas.IQuota, inheritBilling bool, isWithServerCreate bool) (*SDisk, error) {
	lockman.LockObject(ctx, storage)
	defer lockman.ReleaseObject(ctx, storage)

	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)

	diskName := fmt.Sprintf("vdisk_%s_%d", self.Name, time.Now().UnixNano())

	billingType := BILLING_TYPE_POSTPAID
	billingCycle := ""
	if inheritBilling {
		billingType = self.BillingType
		billingCycle = self.BillingCycle
	}

	autoDelete := false
	if storage.IsLocal() || billingType == BILLING_TYPE_PREPAID || isWithServerCreate {
		autoDelete = true
	}
	disk, err := storage.createDisk(diskName, diskConfig, userCred, self.ProjectId, autoDelete, self.IsSystem,
		billingType, billingCycle)

	if err != nil {
		return nil, err
	}

	cancelUsage := SQuota{}
	cancelUsage.Storage = disk.DiskSize
	err = QuotaManager.CancelPendingUsage(ctx, userCred, self.ProjectId, pendingUsage, &cancelUsage)

	return disk, nil
}

func (self *SGuest) ChooseHostStorage(host *SHost, backend string, candidate *schedapi.CandidateDisk) *SStorage {
	log.Errorf("==========candidate %#v", candidate)
	if candidate == nil {
		return self.GetDriver().ChooseHostStorage(host, backend)
	}
	return StorageManager.FetchStorageById(candidate.StorageId)
}

func (self *SGuest) createDiskOnHost(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	host *SHost,
	diskConfig *api.DiskConfig,
	pendingUsage quotas.IQuota,
	inheritBilling bool,
	isWithServerCreate bool,
	candidate *schedapi.CandidateDisk,
) (*SDisk, error) {
	storage := self.ChooseHostStorage(host, diskConfig.Backend, candidate)
	log.Debugf("Choose storage %s:%s for disk %#v", storage.Name, storage.Id, diskConfig)
	if storage == nil {
		return nil, fmt.Errorf("No storage on %s to create disk for %s", host.GetName(), diskConfig.Backend)
	}
	disk, err := self.createDiskOnStorage(ctx, userCred, storage, diskConfig, pendingUsage, inheritBilling, isWithServerCreate)
	if err != nil {
		return nil, err
	}
	// TODO: use scheduler candidate storage
	if len(self.BackupHostId) > 0 {
		backupHost := HostManager.FetchHostById(self.BackupHostId)
		backupStorage := self.GetDriver().ChooseHostStorage(backupHost, diskConfig.Backend)
		diff, err := db.Update(disk, func() error {
			disk.BackupStorageId = backupStorage.Id
			return nil
		})
		if err != nil {
			log.Errorf("Disk save backup storage error")
			return disk, err
		}
		db.OpsLog.LogEvent(disk, db.ACT_UPDATE, diff, userCred)
	}
	err = self.attach2Disk(ctx, disk, userCred, diskConfig.Driver, diskConfig.Cache, diskConfig.Mountpoint)
	return disk, err
}

func (self *SGuest) CreateIsolatedDeviceOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, devs []*api.IsolatedDeviceConfig, pendingUsage quotas.IQuota) error {
	for _, devConfig := range devs {
		err := self.createIsolatedDeviceOnHost(ctx, userCred, host, devConfig, pendingUsage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuest) createIsolatedDeviceOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, devConfig *api.IsolatedDeviceConfig, pendingUsage quotas.IQuota) error {
	lockman.LockClass(ctx, QuotaManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, QuotaManager, self.ProjectId)

	err := IsolatedDeviceManager.attachHostDeviceToGuestByDesc(ctx, self, host, devConfig, userCred)
	if err != nil {
		return err
	}

	cancelUsage := SQuota{IsolatedDevice: 1}
	err = QuotaManager.CancelPendingUsage(ctx, userCred, self.ProjectId, pendingUsage, &cancelUsage)
	return err
}

func (self *SGuest) attachIsolatedDevice(ctx context.Context, userCred mcclient.TokenCredential, dev *SIsolatedDevice) error {
	if len(dev.GuestId) > 0 {
		return fmt.Errorf("Isolated device already attached to another guest: %s", dev.GuestId)
	}
	if dev.HostId != self.HostId {
		return fmt.Errorf("Isolated device and guest are not located in the same host")
	}
	_, err := db.Update(dev, func() error {
		dev.GuestId = self.Id
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_GUEST_ATTACH_ISOLATED_DEVICE, dev.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SGuest) JoinGroups(userCred mcclient.TokenCredential, params *jsonutils.JSONDict) {
	// TODO
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

	guestnics, err := self.GetNetworks("")
	if err != nil {
		log.Errorf("no nics for this server!!! %s", err)
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

func (self *SGuest) LeaveAllGroups(ctx context.Context, userCred mcclient.TokenCredential) {
	groupGuests := make([]SGroupguest, 0)
	q := GroupguestManager.Query()
	err := q.Filter(sqlchemy.Equals(q.Field("guest_id"), self.Id)).All(&groupGuests)
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	for _, gg := range groupGuests {
		gg.Delete(context.Background(), userCred)
		var group SGroup
		gq := GroupManager.Query()
		err := gq.Filter(sqlchemy.Equals(gq.Field("id"), gg.SrvtagId)).First(&group)
		if err != nil {
			log.Errorln(err.Error())
			return
		}
		db.OpsLog.LogDetachEvent(ctx, self, &group, userCred, nil)
	}
}

func (self *SGuest) DetachAllNetworks(ctx context.Context, userCred mcclient.TokenCredential) error {
	// from clouds.models.portmaps import Portmaps
	// Portmaps.delete_guest_network_portmaps(self, user_cred)
	gns, err := self.GetNetworks("")
	if err != nil {
		return err
	}
	return GuestnetworkManager.DeleteGuestNics(ctx, userCred, gns, false)
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
	if query != nil {
		overridePendingDelete = jsonutils.QueryBoolean(query, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(query, "purge", false)
	}
	if (overridePendingDelete || purge) && !db.IsAdminAllowDelete(userCred, self) {
		return false
	}
	return self.IsOwner(userCred) || db.IsAdminAllowDelete(userCred, self)
}

func (self *SGuest) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	overridePendingDelete := false
	purge := false
	if query != nil {
		overridePendingDelete = jsonutils.QueryBoolean(query, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(query, "purge", false)
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

func (self *SGuest) GetDeployConfigOnHost(ctx context.Context, userCred mcclient.TokenCredential, host *SHost, params *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	config := jsonutils.NewDict()

	desc := self.GetDriver().GetJsonDescAtHost(ctx, userCred, self, host)
	config.Add(desc, "desc")

	deploys, err := cmdline.FetchDeployConfigsByJSON(params)
	if err != nil {
		return nil, err
	}

	if len(deploys) > 0 {
		config.Add(jsonutils.Marshal(deploys), "deploys")
	}

	deployAction, _ := params.GetString("deploy_action")
	if len(deployAction) == 0 {
		deployAction = "deploy"
	}

	// resetPasswd := true
	// if deployAction == "deploy" {
	resetPasswd := jsonutils.QueryBoolean(params, "reset_password", true)
	//}

	if resetPasswd {
		config.Add(jsonutils.JSONTrue, "reset_password")
		passwd, _ := params.GetString("password")
		if len(passwd) > 0 {
			config.Add(jsonutils.NewString(passwd), "password")
		}
		keypair := self.getKeypair()
		if keypair != nil {
			config.Add(jsonutils.NewString(keypair.PublicKey), "public_key")
		}
		deletePubKey, _ := params.GetString("delete_public_key")
		if len(deletePubKey) > 0 {
			config.Add(jsonutils.NewString(deletePubKey), "delete_public_key")
		}
	} else {
		config.Add(jsonutils.JSONFalse, "reset_password")
	}

	// add default public keys
	_, adminPubKey, err := sshkeys.GetSshAdminKeypair(ctx)
	if err != nil {
		log.Errorf("fail to get ssh admin public key %s", err)
	}

	_, projPubKey, err := sshkeys.GetSshProjectKeypair(ctx, self.ProjectId)

	if err != nil {
		log.Errorf("fail to get ssh project public key %s", err)
	}

	config.Add(jsonutils.NewString(adminPubKey), "admin_public_key")
	config.Add(jsonutils.NewString(projPubKey), "project_public_key")

	config.Add(jsonutils.NewString(deployAction), "action")

	onFinish := "shutdown"
	if jsonutils.QueryBoolean(params, "auto_start", false) || jsonutils.QueryBoolean(params, "restart", false) {
		onFinish = "none"
	} else if utils.IsInStringArray(self.Status, []string{VM_ADMIN}) {
		onFinish = "none"
	}

	config.Add(jsonutils.NewString(onFinish), "on_finish")

	if deployAction == "create" && !utils.IsInStringArray(self.Hypervisor, []string{HYPERVISOR_KVM, HYPERVISOR_BAREMETAL, HYPERVISOR_CONTAINER, HYPERVISOR_ESXI, HYPERVISOR_XEN}) {
		nets, err := self.GetNetworks("")
		if err != nil || len(nets) == 0 {
			return nil, fmt.Errorf("failed to find network for guest %s: %s", self.Name, err)
		}
		net := nets[0].GetNetwork()
		vpc := net.GetVpc()
		registerVpcId := vpc.ExternalId
		externalVpcId := vpc.ExternalId
		switch self.Hypervisor {
		case HYPERVISOR_ALIYUN, HYPERVISOR_AWS, HYPERVISOR_HUAWEI:
			break
		case HYPERVISOR_QCLOUD, HYPERVISOR_OPENSTACK:
			registerVpcId = "normal"
		case HYPERVISOR_AZURE:
			registerVpcId, externalVpcId = "normal", "normal"
			if strings.HasSuffix(host.Name, "-classic") {
				registerVpcId, externalVpcId = "classic", "classic"
			}
		default:
			return nil, fmt.Errorf("Unknown guest %s hypervisor %s for sync secgroup", self.Name, self.Hypervisor)
		}
		iregion, err := host.GetIRegion()
		if err != nil {
			return nil, fmt.Errorf("failed to get iregion for host %s error: %v", host.Name, err)
		}
		secgroupIds := jsonutils.NewArray()
		secgroups := self.GetSecgroups()
		for i, secgroup := range secgroups {
			secgroupCache := SecurityGroupCacheManager.Register(ctx, userCred, secgroup.Id, registerVpcId, vpc.CloudregionId, vpc.ManagerId)
			if secgroupCache == nil {
				return nil, fmt.Errorf("failed to registor secgroupCache for secgroup: %s(%s), vpc: %s", secgroup.Name, secgroup.Id, vpc.Name)
			}

			externalSecgroupId, err := iregion.SyncSecurityGroup(secgroupCache.ExternalId, externalVpcId, secgroup.Name, secgroup.Description, secgroup.GetSecRules(""))
			if err != nil {
				return nil, fmt.Errorf("SyncSecurityGroup fail %s", err)
			}
			if err := secgroupCache.SetExternalId(userCred, externalSecgroupId); err != nil {
				return nil, fmt.Errorf("failed to set externalId for secgroup %s(%s) externalId %s: error: %v", secgroup.Name, secgroup.Id, externalSecgroupId, err)
			}
			secgroupIds.Add(jsonutils.NewString(externalSecgroupId))
			if i == 0 {
				config.Add(jsonutils.NewString(externalSecgroupId), "desc", "external_secgroup_id")
			}
		}
		config.Add(secgroupIds, "desc", "external_secgroup_ids")
	}
	return config, nil
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

	if len(self.BackupHostId) > 0 {
		if self.HostId == host.Id {
			desc.Set("is_master", jsonutils.JSONTrue)
			desc.Set("host_id", jsonutils.NewString(self.HostId))
		} else if self.BackupHostId == host.Id {
			desc.Set("is_slave", jsonutils.JSONTrue)
			desc.Set("host_id", jsonutils.NewString(self.BackupHostId))
		}
	}

	// isolated devices
	isolatedDevs := IsolatedDeviceManager.generateJsonDescForGuest(self)
	desc.Add(jsonutils.NewArray(isolatedDevs...), "isolated_devices")

	// nics, domain
	jsonNics := make([]jsonutils.JSONObject, 0)

	nics, _ := self.GetNetworks("")

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

	// disks
	jsonDisks := make([]jsonutils.JSONObject, 0)
	disks := self.GetDisks()
	if disks != nil && len(disks) > 0 {
		for _, disk := range disks {
			diskDesc := disk.GetJsonDescAtHost(host)
			jsonDisks = append(jsonDisks, diskDesc)
		}
	}
	desc.Add(jsonutils.NewArray(jsonDisks...), "disks")

	// cdrom
	cdDesc := self.getCdrom().getJsonDesc()
	if cdDesc != nil {
		desc.Add(cdDesc, "cdrom")
	}

	// tenant
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

	if secgroups := self.getSecgroupJson(); len(secgroups) > 0 {
		desc.Add(jsonutils.NewArray(secgroups...), "secgroups")
	}

	/*
		TODO
		srs := self.getSecurityRuleSet()
		if srs.estimatedSinglePortRuleCount() <= options.FirewallFlowCountLimit {
	*/

	rules := self.getSecurityGroupsRules()
	if len(rules) > 0 {
		desc.Add(jsonutils.NewString(rules), "security_rules")
	}
	rules = self.getAdminSecurityRules()
	if len(rules) > 0 {
		desc.Add(jsonutils.NewString(rules), "admin_security_rules")
	}

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

	rules := self.getSecurityGroupsRules()
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

func (self *SGuest) GetSpec(checkStatus bool) *jsonutils.JSONDict {
	if checkStatus {
		if utils.IsInStringArray(self.Status, []string{VM_SCHEDULE_FAILED}) {
			return nil
		}
	}
	spec := jsonutils.NewDict()
	spec.Set("cpu", jsonutils.NewInt(int64(self.VcpuCount)))
	spec.Set("mem", jsonutils.NewInt(int64(self.VmemSize)))

	// get disk spec
	guestdisks := self.GetDisks()
	diskSpecs := jsonutils.NewArray()
	for _, guestdisk := range guestdisks {
		info := guestdisk.ToDiskInfo()
		diskSpec := jsonutils.NewDict()
		diskSpec.Set("size", jsonutils.NewInt(info.Size))
		diskSpec.Set("backend", jsonutils.NewString(info.Backend))
		diskSpec.Set("medium_type", jsonutils.NewString(info.MediumType))
		diskSpec.Set("disk_type", jsonutils.NewString(info.DiskType))
		diskSpecs.Add(diskSpec)
	}
	spec.Set("disk", diskSpecs)

	// get nic spec
	guestnics, _ := self.GetNetworks("")

	nicSpecs := jsonutils.NewArray()
	for _, guestnic := range guestnics {
		nicSpec := jsonutils.NewDict()
		nicSpec.Set("bandwidth", jsonutils.NewInt(int64(guestnic.getBandwidth())))
		t := "int"
		if guestnic.IsExit() {
			t = "ext"
		}
		nicSpec.Set("type", jsonutils.NewString(t))
		nicSpecs.Add(nicSpec)
	}
	spec.Set("nic", nicSpecs)

	// get isolate device spec
	guestgpus := self.GetIsolatedDevices()
	gpuSpecs := jsonutils.NewArray()
	for _, guestgpu := range guestgpus {
		if strings.HasPrefix(guestgpu.DevType, "GPU") {
			gs := guestgpu.GetSpec(false)
			if gs != nil {
				gpuSpecs.Add(gs)
			}
		}
	}
	spec.Set("gpu", gpuSpecs)
	return spec
}

func (manager *SGuestManager) GetSpecIdent(spec *jsonutils.JSONDict) []string {
	cpuCount, _ := spec.Int("cpu")
	memSize, _ := spec.Int("mem")
	memSizeMB, _ := utils.GetSizeMB(fmt.Sprintf("%d", memSize), "M")
	specKeys := []string{
		fmt.Sprintf("cpu:%d", cpuCount),
		fmt.Sprintf("mem:%dM", memSizeMB),
	}

	countKey := func(kf func(*jsonutils.JSONDict) string, dataArray jsonutils.JSONObject) map[string]int64 {
		countMap := make(map[string]int64)
		datas, _ := dataArray.GetArray()
		for _, data := range datas {
			key := kf(data.(*jsonutils.JSONDict))
			if count, ok := countMap[key]; !ok {
				countMap[key] = 1
			} else {
				count++
				countMap[key] = count
			}
		}
		return countMap
	}

	kfuncs := map[string]func(*jsonutils.JSONDict) string{
		"disk": func(data *jsonutils.JSONDict) string {
			backend, _ := data.GetString("backend")
			mediumType, _ := data.GetString("medium_type")
			size, _ := data.Int("size")
			sizeGB, _ := utils.GetSizeGB(fmt.Sprintf("%d", size), "M")
			return fmt.Sprintf("disk:%s_%s_%dG", backend, mediumType, sizeGB)
		},
		"nic": func(data *jsonutils.JSONDict) string {
			typ, _ := data.GetString("type")
			bw, _ := data.Int("bandwidth")
			return fmt.Sprintf("nic:%s_%dM", typ, bw)
		},
		"gpu": func(data *jsonutils.JSONDict) string {
			vendor, _ := data.GetString("vendor")
			model, _ := data.GetString("model")
			return fmt.Sprintf("gpu:%s_%s", vendor, model)
		},
	}

	for sKey, kf := range kfuncs {
		sArrary, err := spec.Get(sKey)
		if err != nil {
			log.Errorf("Get key %s array error: %v", sKey, err)
			continue
		}
		for key, count := range countKey(kf, sArrary) {
			specKeys = append(specKeys, fmt.Sprintf("%sx%d", key, count))
		}
	}
	return specKeys
}

func (self *SGuest) GetTemplateId() string {
	guestdisks := self.GetDisks()
	for _, guestdisk := range guestdisks {
		disk := guestdisk.GetDisk()
		if disk != nil {
			templateId := disk.GetTemplateId()
			if len(templateId) > 0 {
				return templateId
			}
		}
	}
	return ""
}

func (self *SGuest) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SVirtualResourceBase.GetShortDesc(ctx)
	desc.Set("mem", jsonutils.NewInt(int64(self.VmemSize)))
	desc.Set("cpu", jsonutils.NewInt(int64(self.VcpuCount)))

	address := jsonutils.NewString(strings.Join(self.getRealIPs(), ","))
	desc.Set("ip_addr", address)

	if len(self.OsType) > 0 {
		desc.Add(jsonutils.NewString(self.OsType), "os_type")
	}
	if osDist := self.GetMetadata("os_distribution", nil); len(osDist) > 0 {
		desc.Add(jsonutils.NewString(osDist), "os_distribution")
	}
	if osVer := self.GetMetadata("os_version", nil); len(osVer) > 0 {
		desc.Add(jsonutils.NewString(osVer), "os_version")
	}

	templateId := self.GetTemplateId()
	if len(templateId) > 0 {
		desc.Set("template_id", jsonutils.NewString(templateId))
	}
	extBw := self.getBandwidth(true)
	intBw := self.getBandwidth(false)
	if extBw > 0 {
		desc.Set("ext_bandwidth", jsonutils.NewInt(int64(extBw)))
	}
	if intBw > 0 {
		desc.Set("int_bandwidth", jsonutils.NewInt(int64(intBw)))
	}

	if len(self.OsType) > 0 {
		desc.Add(jsonutils.NewString(self.OsType), "os_type")
	}

	if len(self.ExternalId) > 0 {
		desc.Add(jsonutils.NewString(self.ExternalId), "externalId")
	}

	desc.Set("hypervisor", jsonutils.NewString(self.GetHypervisor()))

	host := self.GetHost()

	spec := self.GetSpec(false)
	if self.GetHypervisor() == HYPERVISOR_BAREMETAL {
		if host != nil {
			hostSpec := host.GetSpec(false)
			hostSpecIdent := HostManager.GetSpecIdent(hostSpec)
			spec.Set("host_spec", jsonutils.NewString(strings.Join(hostSpecIdent, "/")))
		}
	}
	if spec != nil {
		desc.Update(spec)
	}

	var billingInfo SCloudBillingInfo

	if host != nil {
		billingInfo.SCloudProviderInfo = host.getCloudProviderInfo()
	}

	if priceKey := self.GetMetadata("price_key", nil); len(priceKey) > 0 {
		billingInfo.PriceKey = priceKey
	}

	billingInfo.SBillingBaseInfo = self.getBillingBaseInfo()

	desc.Update(jsonutils.Marshal(billingInfo))

	return desc
}

func (self *SGuest) saveOsType(userCred mcclient.TokenCredential, osType string) error {
	diff, err := db.Update(self, func() error {
		self.OsType = osType
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	return err
}

func (self *SGuest) SaveDeployInfo(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) {
	// log.Infof("------SaveDeployInfo: %s", data.PrettyString())
	info := make(map[string]interface{})
	if data.Contains("os") {
		osName, _ := data.GetString("os")
		self.saveOsType(userCred, osName)
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
				sqlchemy.IsFalse(guests.Field("pending_deleted"))))).
		Join(networks, sqlchemy.Equals(networks.Field("id"), guestnics.Field("network_id"))).
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
	defer rows.Close()
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

func (manager *SGuestManager) getExpiredPendingDeleteGuests() []SGuest {
	deadline := time.Now().Add(time.Duration(options.Options.PendingDeleteExpireSeconds*-1) * time.Second)

	q := manager.Query()
	q = q.IsTrue("pending_deleted").LT("pending_deleted_at", deadline).Limit(options.Options.PendingDeleteMaxCleanBatchSize)

	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("fetch guests error %s", err)
		return nil
	}

	return guests
}

func (manager *SGuestManager) CleanPendingDeleteServers(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	guests := manager.getExpiredPendingDeleteGuests()
	if guests == nil {
		return
	}
	for i := 0; i < len(guests); i += 1 {
		guests[i].StartDeleteGuestTask(ctx, userCred, "", false, true)
	}
}

func (manager *SGuestManager) getExpiredPrepaidGuests() []SGuest {
	deadline := time.Now().Add(time.Duration(options.Options.PrepaidExpireCheckSeconds*-1) * time.Second)

	q := manager.Query()
	q = q.Equals("billing_type", BILLING_TYPE_PREPAID).LT("expired_at", deadline).Limit(options.Options.ExpiredPrepaidMaxCleanBatchSize)

	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("fetch guests error %s", err)
		return nil
	}

	return guests
}

func (self *SGuest) doExternalSync(ctx context.Context, userCred mcclient.TokenCredential) error {
	host := self.GetHost()
	if host == nil {
		return fmt.Errorf("no host???")
	}
	ihost, iprovider, err := host.GetIHostAndProvider()
	if err != nil {
		return err
	}
	iVM, err := ihost.GetIVMById(self.ExternalId)
	if err != nil {
		return err
	}
	return self.syncWithCloudVM(ctx, userCred, iprovider, host, iVM, "")
}

func (manager *SGuestManager) DeleteExpiredPrepaidServers(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	guests := manager.getExpiredPrepaidGuests()
	if guests == nil {
		return
	}
	for i := 0; i < len(guests); i += 1 {
		// fake delete expired prepaid servers
		if len(guests[i].ExternalId) > 0 {
			err := guests[i].doExternalSync(ctx, userCred)
			if err == nil && guests[i].IsValidPrePaid() {
				continue
			}
		}
		guests[i].SetDisableDelete(userCred, false)
		guests[i].StartDeleteGuestTask(ctx, userCred, "", false, false)
	}
}

func (self *SGuest) GetEip() (*SElasticip, error) {
	return ElasticipManager.getEipForInstance("server", self.Id)
}

func (self *SGuest) GetRealIps() []string {
	return self.getRealIPs()
}

func (self *SGuest) SyncVMEip(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extEip cloudprovider.ICloudEIP, projectId string) compare.SyncResult {
	result := compare.SyncResult{}

	eip, err := self.GetEip()
	if err != nil {
		result.Error(fmt.Errorf("getEip error %s", err))
		return result
	}

	if eip == nil && extEip == nil {
		// do nothing
	} else if eip == nil && extEip != nil {
		// add
		neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, self.getRegion(), projectId)
		if err != nil {
			log.Errorf("getEipByExtEip error %v", err)
			result.AddError(err)
		} else {
			err = neip.AssociateVM(ctx, userCred, self)
			if err != nil {
				log.Errorf("AssociateVM error %v", err)
				result.AddError(err)
			} else {
				result.Add()
			}
		}
	} else if eip != nil && extEip == nil {
		// remove
		err = eip.Dissociate(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	} else {
		// sync
		if eip.IpAddr != extEip.GetIpAddr() {
			// remove then add
			err = eip.Dissociate(ctx, userCred)
			if err != nil {
				// fail to remove
				result.DeleteError(err)
			} else {
				result.Delete()
				neip, err := ElasticipManager.getEipByExtEip(ctx, userCred, extEip, self.getRegion(), projectId)
				if err != nil {
					result.AddError(err)
				} else {
					err = neip.AssociateVM(ctx, userCred, self)
					if err != nil {
						result.AddError(err)
					} else {
						result.Add()
					}
				}
			}
		} else {
			// do nothing
			err := eip.SyncWithCloudEip(ctx, userCred, provider, extEip, projectId)
			if err != nil {
				result.UpdateError(err)
			} else {
				result.Update()
			}
		}
	}

	return result
}

func (self *SGuest) getSecgroupExternalIds(provider *SCloudprovider) []string {
	secgroups := self.GetSecgroups()
	secgroupids := []string{}
	for i := 0; i < len(secgroups); i++ {
		secgroupids = append(secgroupids, secgroups[i].Id)
	}
	q := SecurityGroupCacheManager.Query().Equals("manager_id", provider.Id)
	q = q.Filter(sqlchemy.In(q.Field("secgroup_id"), secgroupids))
	secgroupcaches := []SSecurityGroupCache{}
	if err := db.FetchModelObjects(SecurityGroupCacheManager, q, &secgroupcaches); err != nil {
		log.Errorf("failed to fetch secgroupcaches for provider %s error: %v", provider.Name, err)
		return nil
	}
	externalIds := []string{}
	for i := 0; i < len(secgroupcaches); i++ {
		externalIds = append(externalIds, secgroupcaches[i].ExternalId)
	}
	return externalIds
}

func (self *SGuest) getSecgroupByCache(provider *SCloudprovider, externalId string) (*SSecurityGroup, error) {
	q := SecurityGroupCacheManager.Query().Equals("manager_id", provider.Id).Equals("external_id", externalId)
	cache := SSecurityGroupCache{}
	cache.SetModelManager(SecurityGroupCacheManager)
	count := q.Count()
	if count == 0 {
		return nil, fmt.Errorf("failed find secgroup cache from provider %s externalId %s", provider.Name, externalId)
	}
	if count > 1 {
		return nil, fmt.Errorf("dumplicate secgroup cache for provider %s externalId %s", provider.Name, externalId)
	}
	if err := q.First(&cache); err != nil {
		return nil, err
	}
	return cache.GetSecgroup()
}

func (self *SGuest) SyncVMSecgroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, secgroupIds []string) compare.SyncResult {
	syncResult := compare.SyncResult{}

	secgroupExternalIds := self.getSecgroupExternalIds(provider)

	for _, secgroupId := range secgroupIds {
		if !utils.IsInStringArray(secgroupId, secgroupExternalIds) {
			secgroup, err := self.getSecgroupByCache(provider, secgroupId)
			if err != nil {
				syncResult.AddError(err)
				continue
			}
			if len(self.SecgrpId) == 0 {
				_, err := db.Update(self, func() error {
					self.SecgrpId = secgroup.Id
					return nil
				})
				if err != nil {
					log.Errorf("update guest secgroup error: %v", err)
					syncResult.AddError(err)
				}
			} else {
				if _, err := GuestsecgroupManager.newGuestSecgroup(ctx, userCred, self, secgroup); err != nil {
					log.Errorf("failed to bind secgroup %s for guest %s error: %v", secgroup.Name, self.Name, err)
					syncResult.AddError(err)
				}
			}
			syncResult.Add()
		}
	}
	return syncResult
}

func (self *SGuest) GetIVM() (cloudprovider.ICloudVM, error) {
	if len(self.ExternalId) == 0 {
		msg := fmt.Sprintf("GetIVM: not managed by a provider")
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	host := self.GetHost()
	if host == nil {
		msg := fmt.Sprintf("GetIVM: No valid host")
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	ihost, err := host.GetIHost()
	if err != nil {
		msg := fmt.Sprintf("GetIVM: getihost fail %s", err)
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	return ihost.GetIVMById(self.ExternalId)
}

func (self *SGuest) DeleteEip(ctx context.Context, userCred mcclient.TokenCredential) error {
	eip, err := self.GetEip()
	if err != nil {
		log.Errorf("Delete eip fail for get Eip %s", err)
		return err
	}
	if eip == nil {
		return nil
	}
	if eip.Mode == EIP_MODE_INSTANCE_PUBLICIP {
		err = eip.RealDelete(ctx, userCred)
		if err != nil {
			log.Errorf("Delete eip on delete server fail %s", err)
			return err
		}
	} else {
		err = eip.Dissociate(ctx, userCred)
		if err != nil {
			log.Errorf("Dissociate eip on delete server fail %s", err)
			return err
		}
	}
	return nil
}

func (self *SGuest) SetDisableDelete(userCred mcclient.TokenCredential, val bool) error {
	diff, err := db.Update(self, func() error {
		if val {
			self.DisableDelete = tristate.True
		} else {
			self.DisableDelete = tristate.False
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	logclient.AddSimpleActionLog(self, logclient.ACT_UPDATE, diff, userCred, true)
	return err
}

func (self *SGuest) getDefaultStorageType() string {
	diskCat := self.CategorizeDisks()
	if diskCat.Root != nil {
		rootStorage := diskCat.Root.GetStorage()
		if rootStorage != nil {
			return rootStorage.StorageType
		}
	}
	return STORAGE_LOCAL
}

func (self *SGuest) GetApptags() []string {
	tagsStr := self.GetMetadata("app_tags", nil)
	if len(tagsStr) > 0 {
		return strings.Split(tagsStr, ",")
	}
	return nil
}

func (self *SGuest) ToSchedDesc() *schedapi.ScheduleInput {
	desc := new(schedapi.ScheduleInput)
	config := &schedapi.ServerConfig{
		Name:          self.Name,
		Memory:        self.VmemSize,
		Ncpu:          int(self.VcpuCount),
		ServerConfigs: new(api.ServerConfigs),
	}
	desc.Id = self.Id
	//self.FillGroupSchedDesc(desc)
	self.FillDiskSchedDesc(config.ServerConfigs)
	self.FillNetSchedDesc(config.ServerConfigs)
	if len(self.HostId) > 0 && regutils.MatchUUID(self.HostId) {
		config.HostId = self.HostId
	}
	config.Project = self.ProjectId
	/*tags := self.GetApptags()
	for i := 0; i < len(tags); i++ {
		desc.Set(tags[i], jsonutils.JSONTrue)
	}*/

	config.Hypervisor = self.GetHypervisor()
	desc.ServerConfig = *config
	return desc
}

/*func (self *SGuest) FillGroupSchedDesc(desc *schedapi.ServerConfig) {
	groups := make([]SGroupguest, 0)
	err := GroupguestManager.Query().Equals("guest_id", self.Id).All(&groups)
	if err != nil {
		log.Errorln(err)
		return
	}
	for i := 0; i < len(groups); i++ {
		desc.Set(fmt.Sprintf("srvtag.%d", i),
			jsonutils.NewString(fmt.Sprintf("%s:%s", groups[i].SrvtagId, groups[i].Tag)))
	}
}*/

func (self *SGuest) FillDiskSchedDesc(desc *api.ServerConfigs) {
	guestDisks := make([]SGuestdisk, 0)
	err := GuestdiskManager.Query().Equals("guest_id", self.Id).All(&guestDisks)
	if err != nil {
		log.Errorln("FillDiskSchedDesc: %v", err)
		return
	}
	for i := 0; i < len(guestDisks); i++ {
		diskConf := guestDisks[i].ToDiskConfig()
		// HACK: storage used by self, so earse it
		diskConf.Storage = ""
		desc.Disks = append(desc.Disks, diskConf)
	}
}

func (self *SGuest) FillNetSchedDesc(desc *api.ServerConfigs) {
	guestNetworks := make([]SGuestnetwork, 0)
	err := GuestnetworkManager.Query().Equals("guest_id", self.Id).All(&guestNetworks)
	if err != nil {
		log.Errorln("FillNetSchedDesc: %v", err)
		return
	}
	if desc.Networks == nil {
		desc.Networks = make([]*api.NetworkConfig, 0)
	}
	for i := 0; i < len(guestNetworks); i++ {
		desc.Networks = append(desc.Networks, guestNetworks[i].ToNetworkConfig())
	}
}

func (self *SGuest) GuestDisksHasSnapshot() bool {
	guestDisks := self.GetDisks()
	for i := 0; i < len(guestDisks); i++ {
		if SnapshotManager.GetDiskSnapshotCount(guestDisks[i].DiskId) > 0 {
			return true
		}
	}
	return false
}

func (self *SGuest) OnScheduleToHost(ctx context.Context, userCred mcclient.TokenCredential, hostId string) error {
	err := self.SetHostId(userCred, hostId)
	if err != nil {
		return err
	}

	notes := jsonutils.NewDict()
	notes.Add(jsonutils.NewString(hostId), "host_id")
	db.OpsLog.LogEvent(self, db.ACT_SCHEDULE, notes, userCred)

	return nil
}

func (guest *SGuest) AllowGetDetailsTasks(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, guest, "tasks")
}

func (guest *SGuest) GetDetailsTasks(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	since := time.Time{}
	if query.Contains("since") {
		since, _ = query.GetTime("since")
	}
	var isOpen *bool = nil
	if query.Contains("is_open") {
		isOpenVal, _ := query.Bool("is_open")
		isOpen = &isOpenVal
	}
	q := taskman.TaskManager.QueryTasksOfObject(guest, since, isOpen)
	objs, err := db.Query2List(taskman.TaskManager, ctx, userCred, q, query)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewArray(objs...), "tasks")
	return ret, nil
}

func (guest *SGuest) GetDynamicConditionInput() *jsonutils.JSONDict {
	return guest.ToSchedDesc().ToConditionInput()
}
