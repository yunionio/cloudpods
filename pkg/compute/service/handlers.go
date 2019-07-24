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

package service

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/capabilities"
	"yunion.io/x/onecloud/pkg/compute/misc"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/specs"
	"yunion.io/x/onecloud/pkg/compute/sshkeys"
	"yunion.io/x/onecloud/pkg/compute/usages"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()

	db.RegistUserCredCacheUpdater()

	db.AddProjectResourceCountHandler("", app)

	quotas.AddQuotaHandler(&models.QuotaManager.SQuotaBaseManager, "", app)

	usages.AddUsageHandler("", app)
	capabilities.AddCapabilityHandler("", app)
	specs.AddSpecHandler("", app)
	sshkeys.AddSshKeysHandler("", app)
	taskman.AddTaskHandler("", app)
	misc.AddMiscHandler("", app)

	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		db.UserCacheManager,
		db.TenantCacheManager,
		db.SharedResourceManager,
		models.GuestcdromManager,
		models.NetInterfaceManager,
		models.VCenterManager,

		models.QuotaManager,
		models.QuotaUsageManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
		db.Metadata,
		models.BucketManager,
		models.CloudaccountManager,
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
		models.SecurityGroupCacheManager,
		models.SecurityGroupRuleManager,
		// models.VCenterManager,
		models.DnsRecordManager,
		models.ElasticipManager,
		models.NatGatewayManager,
		models.NatDTableManager,
		models.NatSTableManager,
		models.SnapshotManager,
		models.SnapshotPolicyManager,
		models.BaremetalagentManager,
		models.LoadbalancerManager,
		models.LoadbalancerListenerManager,
		models.LoadbalancerListenerRuleManager,
		models.LoadbalancerBackendGroupManager,
		models.LoadbalancerBackendManager,
		models.LoadbalancerCertificateManager,
		models.LoadbalancerAclManager,
		models.LoadbalancerAgentManager,
		models.LoadbalancerClusterManager,
		models.RouteTableManager,

		models.SchedpolicyManager,
		models.DynamicschedtagManager,

		models.ServerSkuManager,
		models.ExternalProjectManager,
		models.NetworkInterfaceManager,
		models.NetworkinterfacenetworkManager,
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
		models.StorageschedtagManager,
		models.NetworkschedtagManager,
		models.GuestnetworkManager,
		models.GuestsecgroupManager,
		models.LoadbalancernetworkManager,
		models.GuestdiskManager,
		models.GroupnetworkManager,
		models.GroupguestManager,
		models.StoragecachedimageManager,
		models.CloudproviderRegionManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewJointModelHandler(manager)
		dispatcher.AddJointModelDispatcher("", app, handler)
	}
}
