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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/proxy"
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

	db.AddScopeResourceCountHandler("", app)

	quotas.AddQuotaHandler(&models.QuotaManager.SQuotaBaseManager, "", app)
	quotas.AddQuotaHandler(&models.RegionQuotaManager.SQuotaBaseManager, "", app)
	quotas.AddQuotaHandler(&models.ZoneQuotaManager.SQuotaBaseManager, "", app)
	quotas.AddQuotaHandler(&models.ProjectQuotaManager.SQuotaBaseManager, "", app)
	quotas.AddQuotaHandler(&models.DomainQuotaManager.SQuotaBaseManager, "", app)
	quotas.AddQuotaHandler(&models.InfrasQuotaManager.SQuotaBaseManager, "", app)

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
		models.QuotaPendingUsageManager,
		models.ZoneQuotaManager,
		models.ZoneUsageManager,
		models.ZonePendingUsageManager,
		models.RegionQuotaManager,
		models.RegionUsageManager,
		models.RegionPendingUsageManager,
		models.ProjectQuotaManager,
		models.ProjectUsageManager,
		models.ProjectPendingUsageManager,
		models.DomainQuotaManager,
		models.DomainUsageManager,
		models.DomainPendingUsageManager,
		models.InfrasQuotaManager,
		models.InfrasUsageManager,
		models.InfrasPendingUsageManager,

		models.GroupguestManager,

		models.CloudproviderCapabilityManager,

		models.ScalingTimerManager,
		models.ScalingAlarmManager,
		models.ScalingGroupGuestManager,
		models.ScalingGroupNetworkManager,

		models.ScheduledTaskLabelManager,
		models.DnsRecordSetTrafficPolicyManager,
	} {
		db.RegisterModelManager(manager)
	}

	for _, manager := range []db.IModelManager{
		db.OpsLog,
		db.Metadata,

		proxy.ProxySettingManager,

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
		models.NetworkAddressManager,
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
		models.NatDEntryManager,
		models.NatSEntryManager,
		models.InstanceSnapshotManager,
		models.SnapshotManager,
		models.SnapshotPolicyManager,
		models.SnapshotPolicyCacheManager,
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
		models.CachedLoadbalancerAclManager,
		models.CachedLoadbalancerCertificateManager,
		models.HuaweiCachedLbManager,
		models.HuaweiCachedLbbgManager,
		models.AwsCachedLbManager,
		models.AwsCachedLbbgManager,
		models.QcloudCachedLbManager,
		models.QcloudCachedLbbgManager,
		models.OpenstackCachedLbManager,
		models.OpenstackCachedLbbgManager,
		models.RouteTableManager,
		models.RouteTableAssociationManager,
		models.RouteTableRouteSetManager,
		models.InterVpcNetworkRouteSetManager,

		models.SchedpolicyManager,
		models.DynamicschedtagManager,

		models.ServerSkuManager,
		models.ExternalProjectManager,
		models.NetworkInterfaceManager,
		models.DBInstanceManager,
		models.DBInstanceBackupManager,
		models.DBInstanceParameterManager,
		models.DBInstanceDatabaseManager,
		models.DBInstanceAccountManager,
		models.DBInstancePrivilegeManager,
		models.DBInstanceSkuManager,

		models.ElasticcacheManager,
		models.ElasticcacheAclManager,
		models.ElasticcacheAccountManager,
		models.ElasticcacheParameterManager,
		models.ElasticcacheBackupManager,
		models.ElasticcacheSkuManager,
		models.GlobalVpcManager,

		models.GuestTemplateManager,
		models.ServiceCatalogManager,
		models.CloudproviderQuotaManager,

		models.ScalingGroupManager,
		models.ScalingPolicyManager,
		models.ScalingActivityManager,
		models.PolicyDefinitionManager,
		models.PolicyAssignmentManager,

		models.ScheduledTaskManager,
		models.ScheduledTaskActivityManager,

		models.DnsZoneManager,
		models.DnsZoneCacheManager,
		models.DnsRecordSetManager,
		models.DnsTrafficPolicyManager,

		models.VpcPeeringConnectionManager,
		models.InterVpcNetworkManager,
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
		models.CloudproviderschedtagManager,
		models.ZoneschedtagManager,
		models.CloudregionschedtagManager,
		models.GuestnetworkManager,
		models.GuestsecgroupManager,
		models.LoadbalancernetworkManager,
		models.GuestdiskManager,
		models.GroupnetworkManager,
		models.StoragecachedimageManager,
		models.CloudproviderRegionManager,
		models.DBInstanceNetworkManager,
		models.NetworkinterfacenetworkManager,
		models.SnapshotPolicyDiskManager,
		models.InstanceSnapshotJointManager,
		models.DnsZoneVpcManager,
		models.DBInstanceSecgroupManager,
		models.ElasticcachesecgroupManager,
		models.InterVpcNetworkVpcManager,
	} {
		db.RegisterModelManager(manager)
		handler := db.NewJointModelHandler(manager)
		dispatcher.AddJointModelDispatcher("", app, handler)
	}
}
