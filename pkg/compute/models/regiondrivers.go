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

package models

import (
	"context"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type IRegionDriver interface {
	GetProvider() string

	IElasticcacheDriver
	IElasticcacheAccount
	IElasticcacheAcl
	IElasticcacheBackup
	IDBInstanceDriver

	ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *api.LoadbalancerCreateInput) (*api.LoadbalancerCreateInput, error)
	ValidateDeleteLoadbalancerCondition(ctx context.Context, lb *SLoadbalancer) error
	RequestCreateLoadbalancerInstance(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, input *api.LoadbalancerCreateInput, task taskman.ITask) error
	RequestDeleteLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error
	RequestStartLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error
	RequestStopLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error
	RequestSyncstatusLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error
	RequestRemoteUpdateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, replaceTags bool, task taskman.ITask) error

	ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *SCachedLoadbalancerAcl, task taskman.ITask) error
	RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *SCachedLoadbalancerAcl, task taskman.ITask) error
	RequestSyncLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *SCachedLoadbalancerAcl, task taskman.ITask) error

	IsCertificateBelongToRegion() bool
	ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateUpdateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *SCachedLoadbalancerCertificate, task taskman.ITask) error
	RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *SCachedLoadbalancerCertificate, task taskman.ITask) error

	ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lb *SLoadbalancer, backends []cloudprovider.SLoadbalancerBackend) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error
	RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *SLoadbalancerBackendGroup, task taskman.ITask) error
	ValidateDeleteLoadbalancerBackendGroupCondition(ctx context.Context, lbbb *SLoadbalancerBackendGroup) error
	RequestSyncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, task taskman.ITask) error
	RequestPullRegionLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localRegion *SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *SSyncRange) error
	RequestPullLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults SSyncResultSet, provider *SCloudprovider, localLoadbalancer *SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *SSyncRange) error
	GetBackendStatusForAdd() []string

	ValidateCreateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendType string, lb *SLoadbalancer, backendGroup *SLoadbalancerBackendGroup, backend db.IModel) (*jsonutils.JSONDict, error)
	ValidateUpdateLoadbalancerBackendData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lbbg *SLoadbalancerBackendGroup) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *SLoadbalancerBackend, task taskman.ITask) error
	RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *SLoadbalancerBackend, task taskman.ITask) error
	RequestSyncLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *SLoadbalancerBackend, task taskman.ITask) error

	ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, lb *SLoadbalancer, backendGroup db.IModel) (*jsonutils.JSONDict, error)
	ValidateUpdateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lblist *SLoadbalancerListener, backendGroup db.IModel) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, task taskman.ITask) error
	RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, task taskman.ITask) error
	RequestStartLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, task taskman.ITask) error
	RequestStopLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, task taskman.ITask) error
	RequestSyncstatusLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, task taskman.ITask) error
	RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, task taskman.ITask) error

	IsSupportLoadbalancerListenerRuleRedirect() bool
	ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error)
	ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *SLoadbalancerListenerRule, task taskman.ITask) error
	RequestDeleteLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *SLoadbalancerListenerRule, task taskman.ITask) error

	ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, input api.VpcCreateInput) (api.VpcCreateInput, error)
	IsVpcCreateNeedInputCidr() bool
	GetEipDefaultChargeType() string
	ValidateEipChargeType(chargeType string) error
	ValidateCreateEipData(ctx context.Context, userCred mcclient.TokenCredential, input *api.SElasticipCreateInput) error
	RequestCreateVpc(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, vpc *SVpc, task taskman.ITask) error
	RequestDeleteVpc(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, vpc *SVpc, task taskman.ITask) error

	// Region Driver Snapshot Policy Apis
	//ValidateCreateSnapshotPolicyData(context.Context, mcclient.TokenCredential, *compute.SSnapshotPolicyCreateInput, mcclient.IIdentityProvider, *jsonutils.JSONDict) error
	RequestUpdateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, sp *SSnapshotPolicy, input cloudprovider.SnapshotPolicyInput, task taskman.ITask) error
	RequestApplySnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, task taskman.ITask, disk *SDisk, sp *SSnapshotPolicy, data jsonutils.JSONObject) error
	RequestCancelSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, task taskman.ITask, disk *SDisk, sp *SSnapshotPolicy, data jsonutils.JSONObject) error
	RequestPreSnapshotPolicyApply(ctx context.Context, userCred mcclient.TokenCredential, task taskman.ITask, disk *SDisk, sp *SSnapshotPolicy, data jsonutils.JSONObject) error

	// Region Driver Snapshot Policy joint Disk Apis
	ValidateCreateSnapshopolicyDiskData(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, snapshotPolicy *SSnapshotPolicy) error

	// Region Driver Snapshot Apis
	ValidateSnapshotDelete(ctx context.Context, snapshot *SSnapshot) error
	ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, storage *SStorage, input *api.SnapshotCreateInput) error
	RequestCreateSnapshot(ctx context.Context, snapshot *SSnapshot, task taskman.ITask) error
	RequestDeleteSnapshot(ctx context.Context, snapshot *SSnapshot, task taskman.ITask) error
	SnapshotIsOutOfChain(disk *SDisk) bool
	GetDiskResetParams(snapshot *SSnapshot) *jsonutils.JSONDict
	OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, snapshot *SSnapshot, data jsonutils.JSONObject) error
	OnSnapshotDelete(ctx context.Context, snapshot *SSnapshot, task taskman.ITask, data jsonutils.JSONObject) error

	RequestSyncDiskBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, backup *SDiskBackup, task taskman.ITask) error
	RequestCreateBackup(ctx context.Context, backup *SDiskBackup, snapshotId string, task taskman.ITask) error
	RequestDeleteBackup(ctx context.Context, backup *SDiskBackup, task taskman.ITask) error
	RequestCreateInstanceBackup(ctx context.Context, guest *SGuest, ib *SInstanceBackup, task taskman.ITask, params *jsonutils.JSONDict) error
	RequestDeleteInstanceBackup(ctx context.Context, ib *SInstanceBackup, task taskman.ITask) error
	RequestSyncInstanceBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, ib *SInstanceBackup, task taskman.ITask) error
	RequestSyncBackupStorageStatus(ctx context.Context, userCred mcclient.TokenCredential, bs *SBackupStorage, task taskman.ITask) error

	RequestCreateInstanceSnapshot(ctx context.Context, guest *SGuest, isp *SInstanceSnapshot, task taskman.ITask, params *jsonutils.JSONDict) error
	RequestDeleteInstanceSnapshot(ctx context.Context, isp *SInstanceSnapshot, task taskman.ITask) error
	RequestResetToInstanceSnapshot(ctx context.Context, guest *SGuest, isp *SInstanceSnapshot, task taskman.ITask, params *jsonutils.JSONDict) error
	RequestPackInstanceBackup(ctx context.Context, ib *SInstanceBackup, task taskman.ITask, packageName string) error
	RequestUnpackInstanceBackup(ctx context.Context, ib *SInstanceBackup, task taskman.ITask, packageName string, metadataOnly bool) error

	RequestCacheSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, vpc *SVpc, secgroup *SSecurityGroup, classic bool, removeProjectId string, task taskman.ITask) error
	RequestSyncSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, vpcId string, vpc *SVpc, secgroup *SSecurityGroup, removeProjectId, service string, skipSyncRule bool) (string, error)
	GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule
	GenerateSecurityGroupName(name string) string
	IsAllowSecurityGroupNameRepeat() bool
	GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule
	GetSecurityGroupRuleMaxPriority() int
	GetSecurityGroupRuleMinPriority() int
	IsOnlySupportAllowRules() bool
	IsPeerSecgroupWithSameProject() bool
	IsSupportPeerSecgroup() bool
	IsSupportClassicSecurityGroup() bool
	IsSecurityGroupBelongVpc() bool
	IsVpcBelongGlobalVpc() bool
	IsSecurityGroupBelongGlobalVpc() bool //安全组子账号范围内可用
	GetDefaultSecurityGroupVpcId() string
	GetSecurityGroupVpcId(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, host *SHost, vpc *SVpc, classic bool) (string, error)
	GetSecurityGroupPublicScope(service string) rbacutils.TRbacScope

	IsSupportedBillingCycle(bc billing.SBillingCycle, resource string) bool
	GetSecgroupVpcid(vpcId string) string

	RequestSyncDiskStatus(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, task taskman.ITask) error
	RequestSyncSnapshotStatus(ctx context.Context, userCred mcclient.TokenCredential, snapshot *SSnapshot, task taskman.ITask) error
	RequestSyncNatGatewayStatus(ctx context.Context, userCred mcclient.TokenCredential, natgateway *SNatGateway, task taskman.ITask) error
	RequestSyncBucketStatus(ctx context.Context, userCred mcclient.TokenCredential, bucket *SBucket, task taskman.ITask) error
	RequestSyncDBInstanceBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, backup *SDBInstanceBackup, task taskman.ITask) error

	RequestCreateNetwork(ctx context.Context, userCred mcclient.TokenCredential, network *SNetwork) error

	ValidateCreateCdnData(ctx context.Context, userCred mcclient.TokenCredential, input api.CDNDomainCreateInput) (api.CDNDomainCreateInput, error)
}

type IDBInstanceDriver interface {
	ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.DBInstanceCreateInput, skus []SDBInstanceSku, network *SNetwork) (api.DBInstanceCreateInput, error)
	ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *SDBInstance, input api.DBInstanceAccountCreateInput) (api.DBInstanceAccountCreateInput, error)
	ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *SDBInstance, input api.DBInstanceDatabaseCreateInput) (api.DBInstanceDatabaseCreateInput, error)
	ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *SDBInstance, input api.DBInstanceBackupCreateInput) (api.DBInstanceBackupCreateInput, error)
	ValidateDBInstanceAccountPrivilege(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, account string, privilege string) error
	ValidateResetDBInstancePassword(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, account string) error

	RequestCreateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *SDBInstance, task taskman.ITask) error
	RequestCreateDBInstanceFromBackup(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *SDBInstance, task taskman.ITask) error
	RequestCreateDBInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, backup *SDBInstanceBackup, task taskman.ITask) error
	RequestChangeDBInstanceConfig(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, input *api.SDBInstanceChangeConfigInput, task taskman.ITask) error

	IsSupportedDBInstance() bool
	IsSupportedDBInstanceAutoRenew() bool
	IsSupportDBInstancePublicConnection() bool
	IsSupportKeepDBInstanceManualBackup() bool

	InitDBInstanceUser(ctx context.Context, dbinstance *SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error
	GetRdsSupportSecgroupCount() int

	ValidateDBInstanceRecovery(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, backup *SDBInstanceBackup, input api.SDBInstanceRecoveryConfigInput) error

	RequestRemoteUpdateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, replaceTags bool, task taskman.ITask) error
	RequestSyncRdsSecurityGroups(ctx context.Context, userCred mcclient.TokenCredential, rds *SDBInstance, task taskman.ITask) error

	INatGatewayDriver

	IElasticIpDriver
	INasDriver

	IWafDriver
}

type IWafDriver interface {
	ValidateCreateWafInstanceData(ctx context.Context, userCred mcclient.TokenCredential, input api.WafInstanceCreateInput) (api.WafInstanceCreateInput, error)
	ValidateCreateWafRuleData(ctx context.Context, userCred mcclient.TokenCredential, waf *SWafInstance, input api.WafRuleCreateInput) (api.WafRuleCreateInput, error)
}

type INasDriver interface {
	RequestSyncAccessGroup(ctx context.Context, userCred mcclient.TokenCredential, fs *SFileSystem, mt *SMountTarget, ag *SAccessGroup, task taskman.ITask) error
	IsSupportedNas() bool
}

// NAT
type INatGatewayDriver interface {
	IsSupportedNatGateway() bool
	IsSupportedNatAutoRenew() bool
	RequestAssociateEipForNAT(ctx context.Context, userCred mcclient.TokenCredential, nat *SNatGateway, eip *SElasticip, task taskman.ITask) error
	ValidateCreateNatGateway(ctx context.Context, userCred mcclient.TokenCredential, input api.NatgatewayCreateInput) (api.NatgatewayCreateInput, error)
	OnNatEntryDeleteComplete(ctx context.Context, userCred mcclient.TokenCredential, eip *SElasticip) error
}

type IElasticcacheDriver interface {
	IsSupportedElasticcache() bool
	// capability
	IsSupportedElasticcacheSecgroup() bool
	IsSupportedElasticcacheAutoRenew() bool
	GetMaxElasticcacheSecurityGroupCount() int

	AllowCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, elasticcache *SElasticcache) error
	AllowUpdateElasticcacheAuthMode(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, elasticcache *SElasticcache) error
	ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ElasticcacheCreateInput) (*api.ElasticcacheCreateInput, error)
	ValidateCreateElasticcacheAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateCreateElasticcacheAclData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateCreateElasticcacheBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	RequestCreateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask, data *jsonutils.JSONDict) error
	RequestCreateElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheAccount *SElasticcacheAccount, task taskman.ITask) error
	RequestCreateElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheAcl *SElasticcacheAcl, task taskman.ITask) error
	RequestCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheBackup *SElasticcacheBackup, task taskman.ITask) error
	RequestRenewElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, bc billing.SBillingCycle) (time.Time, error)
	RequestElasticcacheSetAutoRenew(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, autoRenew bool, task taskman.ITask) error
	RequestRestartElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestSyncElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestSyncSecgroupsForElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *SElasticcache, task taskman.ITask) error
	RequestDeleteElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestDeleteElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, ea *SElasticcacheAccount, task taskman.ITask) error
	RequestDeleteElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, ea *SElasticcacheAcl, task taskman.ITask) error
	RequestDeleteElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, eb *SElasticcacheBackup, task taskman.ITask) error
	RequestSetElasticcacheMaintainTime(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheChangeSpec(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestUpdateElasticcacheAuthMode(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestUpdateElasticcacheSecgroups(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheSetMaintainTime(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheAllocatePublicConnection(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheReleasePublicConnection(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheFlushInstance(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheUpdateInstanceParameters(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheUpdateBackupPolicy(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error

	RequestSyncElasticcacheStatus(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error

	RequestRemoteUpdateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, replaceTags bool, task taskman.ITask) error
}

type IElasticcacheAccount interface {
	RequestElasticcacheAccountResetPassword(ctx context.Context, userCred mcclient.TokenCredential, ea *SElasticcacheAccount, task taskman.ITask) error
}

type IElasticcacheAcl interface {
	RequestElasticcacheAclUpdate(ctx context.Context, userCred mcclient.TokenCredential, ea *SElasticcacheAcl, task taskman.ITask) error
}

type IElasticcacheBackup interface {
	RequestElasticcacheBackupRestoreInstance(ctx context.Context, userCred mcclient.TokenCredential, ea *SElasticcacheBackup, task taskman.ITask) error
}

type IElasticIpDriver interface {
	RequestAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, eip *SElasticip, input api.ElasticipAssociateInput, obj db.IStatusStandaloneModel, task taskman.ITask) error
}

var regionDrivers map[string]IRegionDriver

func init() {
	regionDrivers = make(map[string]IRegionDriver)
}

func RegisterRegionDriver(driver IRegionDriver) {
	regionDrivers[driver.GetProvider()] = driver
}

func GetRegionDriver(provider string) IRegionDriver {
	driver, ok := regionDrivers[provider]
	if ok {
		return driver
	}
	log.Fatalf("Unsupported provider %s", provider)
	return nil
}
