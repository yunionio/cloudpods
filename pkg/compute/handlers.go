package compute

import (
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/appsrv"
	"github.com/yunionio/onecloud/pkg/appsrv/dispatcher"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/quotas"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/onecloud/pkg/compute/usages"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	quotas.AddQuotaHandler(models.QuotaManager, "", app)
	usages.AddUsageHandler("", app)

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
