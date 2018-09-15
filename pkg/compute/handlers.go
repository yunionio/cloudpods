package compute

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/capabilities"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/specs"
	"yunion.io/x/onecloud/pkg/compute/usages"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	quotas.AddQuotaHandler(models.QuotaManager, "", app)
	usages.AddUsageHandler("", app)
	capabilities.AddCapabilityHandler("", app)
	specs.AddSpecHandler("", app)

	taskman.AddTaskHandler("", app)

	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		db.UserCacheManager,
		db.TenantCacheManager,
		db.Metadata,
		models.GuestcdromManager,
		models.NetInterfaceManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
		models.CloudproviderManager,
		models.CloudregionManager,
		models.ZoneManager,
		models.VpcManager,
		models.WireManager,
		models.StorageManager,
		models.StoragecacheManager,
		models.CachedimageManager,
		models.HostManager,
		models.SchedtagManager,
		models.GuestManager,
		models.GroupManager,
		models.DiskManager,
		models.NetworkManager,
		models.ReservedipManager,
		models.KeypairManager,
		models.IsolatedDeviceManager,
		models.SecurityGroupManager,
		models.SecurityGroupRuleManager,
		models.VCenterManager,
		models.DnsRecordManager,
		models.ElasticipManager,
		models.SnapshotManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler)
	}

	for _, manager := range []db.IJointModelManager{
		models.HostwireManager,
		models.HostnetworkManager,
		models.HoststorageManager,
		models.HostschedtagManager,
		models.GuestnetworkManager,
		models.GuestdiskManager,
		models.GroupnetworkManager,
		models.GroupguestManager,
		models.StoragecachedimageManager,
	} {
		db.RegisterModelManager(manager)
		log.Infof("Register handler %s", manager.KeywordPlural())
		handler := db.NewJointModelHandler(manager)
		dispatcher.AddJointModelDispatcher("", app, handler)
	}
}
