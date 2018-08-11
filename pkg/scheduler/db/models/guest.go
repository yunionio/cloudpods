package models

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"

	"yunion.io/x/pkg/util/sets"
)

const (
	GuestResourceName = "server"

	VmInit           = "init"
	VmUnknown        = "unknown"
	VmSchedule       = "schedule"
	VmScheduleFailed = "schedule_fail"
	VmCreateNetwork  = "network"
	VmNetworkFailed  = "net_fail"
	VmCreateDisk     = "disk"
	VmDiskFailed     = "disk_fail"
	VmStartDeploy    = "start_deploy"
	VmDeploying      = "deploying"
	VmDeployFailed   = "deploy_fail"
	VmReady          = "ready"
	VmStartStart     = "start_start"
	VmStarting       = "starting"
	VmStartFailed    = "start_fail"
	VmStartRestart   = "start_restart"
	VmRunning        = "running"
	VmStartStop      = "start_stop"
	VmStopping       = "stopping"
	VmStopFailed     = "stop_fail"

	VmStartSuspend  = "start_suspend"
	VmSuspending    = "suspending"
	VmSuspend       = "suspend"
	VmSuspendFailed = "suspend_failed"

	VmReset       = "reset"
	VmStartDelete = "start_delete"
	VmDeleteFail  = "delete_fail"
	VmDeleting    = "deleting"

	VmStartMigrate  = "start_migrate"
	VmMigrating     = "migrating"
	VmMigrateFailed = "migrate_failed"

	VmDiskMigrating     = "disk_migrating"
	VmDiskMigrateFailed = "disk_migrate_fail"

	VmChangeFlavor     = "change_flavor"
	VmChangeFlavorFail = "change_flavor_fail"

	VmRebuildRoot     = "rebuild_root"
	VmRebuildRootFail = "rebuild_root_fail"

	VmRebuildDisk     = "rebuild_disk"
	VmRebuildDiskFail = "rebuild_disk_fail"

	VmBlockStream = "block_stream"

	VmStartSnapshot  = "snapshot_start"
	VmSnapshot       = "snapshot"
	VmSnapshotSucc   = "snapshot_succ"
	VmSnapshotFailed = "snapshot_failed"

	VmSyncConfig     = "sync_config"
	VmSyncConfigFail = "sync_config_failed"

	VmResizeDisk     = "resize_disk"
	VmStartSaveDisk  = "start_save_disk"
	VmSaveDisk       = "save_disk"
	VmSaveDiskFailed = "save_disk_failed"

	VmRestoringSnapshot = "restoring_snapshot"

	VmRestoreDisk   = "restore_disk"
	VmRestoreState  = "restore_state"
	VmRestoreFailed = "restore_failed"

	VmRemoveStatefile = "remove_state"

	VmHotplugCPUMEM = "hotplug_cpu_mem"

	VmAdmin = "admin"

	ShutdownStop      = "stop"
	ShutdownTerminate = "terminate"

	HostTypeHost      = "host"
	HostTypeBaremetal = "baremetal"

	GuestTypeVm        = "vm"
	GuestTypeContainer = "container"

	QGAStatusUnknown     = "unknown"
	QGAStatusStop        = "stop"
	QGAStatusStarting    = "starting"
	QGAStatusStartFailed = "start_failed"
	QGAStatusRunning     = "running"
	QGAStatusCrashed     = "crashed"
)

var (
	VmRunningStatus   = sets.NewString(VmStartStart, VmStarting, VmRunning, VmStopFailed, VmBlockStream)
	VmCreatingStatus  = sets.NewString(VmCreateNetwork, VmCreateDisk, VmStartDeploy, VmDeploying)
	GuestExtraFeature = sets.NewString("kvm", "storage_type")
)

type Guest struct {
	VirtualResourceModel
	VCPUCount        int64  `json:"vcpu_count" gorm:"column:vcpu_count;type:tinyint64(4);not null"`
	VMemSize         int64  `json:"vmem_size" gorm:"column:vmem_size;type:int64(11);not null"`
	DimmSlots        string `json:"dimm_slots,omitempty" gorm:"column:dimm_slots;type:text"`
	BootOrder        string `json:"boot_order,omitempty" gorm:"column:boot_order"`
	DisableDelete    bool   `json:"disable_delete" gorm:"column:disable_delete"`
	ShutdownBehavior string `json:"shutdown_behavior,omitempty" gorm:"column:shutdown_behavior"`
	KeypairID        string `json:"keypair_id,omitempty" gorm:"column:keypair_id"`
	HostID           string `json:"host_id,omitempty" gorm:"column:host_id"`
	VNCPort          int64  `json:"vnc_port,omitempty" gorm:"column:vnc_port"`
	VGA              string `json:"vga" gorm:"column:vga"`
	FlavorID         string `json:"flavor_id,omitempty" gorm:"column:flavor_id"`
	SecgrpID         string `json:"secgrp_id,omitempty" gorm:"column:secgrp_id"`
	AdminSecgrpID    string `json:"admin_secgrp_id,omitempty" gorm:"column:admin_secgrp_id"`
	VrouterID        string `json:"vrouter_id,omitempty" gorm:"column:vrouter_id"`
	HostType         string `json:"host_type,omitempty" gorm:"column:host_type"`
	PreferZoneID     string `json:"prefer_zone_id,omitempty" gorm:"column:prefer_zone_id"`
	GuestType        string `json:"guest_type,omitempty" gorm:"column:guest_type"`
	QGAStatus        string `json:"qga_status" gorm:"column:qga_status"`
}

func (g Guest) TableName() string {
	return guestsTable
}

func (g Guest) String() string {
	s, _ := JsonString(g)
	return s
}

func (g Guest) DisksQuery(diskFormat ...string) *gorm.DB {
	q := GuestDisks.DB().Table(guestDiskTable).
		Where(map[string]interface{}{
			"deleted":  false,
			"guest_id": g.ID,
		})
	if len(diskFormat) != 0 {
		joinStr := fmt.Sprintf("JOIN %s on %s.id = %s.disk_id AND %s.disk_format = ?", disksTable, disksTable, guestDiskTable, disksTable)
		q.Joins(joinStr, diskFormat[0])
	}

	return q
}

func (g Guest) Disks(diskFormat ...string) *gorm.DB {
	q := g.DisksQuery(diskFormat...)
	q.Order(fmt.Sprintf("%s.index", g.TableName()))
	return q
}

func (g Guest) DiskSize(onlyLocal bool) (int64, error) {
	var size int64
	q := g.Disks()
	disks := []GuestDisk{}
	err := q.Scan(&disks).Error
	if err != nil {
		return 0, err
	}
	for _, gstDisk := range disks {
		disk, err := gstDisk.Disk()
		if err != nil {
			return 0, err
		}
		isLocal, err := disk.IsLocal()
		if err != nil {
			return 0, err
		}
		if !onlyLocal || isLocal {
			size += disk.DiskSize
		}
	}
	return size, nil
}

func (g Guest) IsRunning() bool {
	return VmRunningStatus.Has(g.Status)
}

func (g Guest) IsCreating() bool {
	return VmCreatingStatus.Has(g.Status)
}

func (g Guest) IsGuestFakeDeleted() bool {
	return strings.HasSuffix(g.Name, "_deleted")
}

func NewGuestResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Guest{}
	}
	models := func() interface{} {
		guests := []Guest{}
		return &guests
	}

	return newResource(db, guestsTable, model, models)
}
