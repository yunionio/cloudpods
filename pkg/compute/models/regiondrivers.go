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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type IRegionDriver interface {
	GetProvider() string

	IElasticcacheDriver
	IElasticcacheAccount
	IElasticcacheAcl
	IElasticcacheBackup

	ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateDeleteLoadbalancerCondition(ctx context.Context, lb *SLoadbalancer) error
	RequestCreateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error
	RequestDeleteLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error
	RequestStartLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error
	RequestStopLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error
	RequestSyncstatusLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *SLoadbalancer, task taskman.ITask) error

	ValidateCreateLoadbalancerAclData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *SCachedLoadbalancerAcl, task taskman.ITask) error
	RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *SCachedLoadbalancerAcl, task taskman.ITask) error
	RequestSyncLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *SCachedLoadbalancerAcl, task taskman.ITask) error

	ValidateCreateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateUpdateLoadbalancerCertificateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *SCachedLoadbalancerCertificate, task taskman.ITask) error
	RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *SCachedLoadbalancerCertificate, task taskman.ITask) error

	ValidateCreateLoadbalancerBackendGroupData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, lb *SLoadbalancer, backends []cloudprovider.SLoadbalancerBackend) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error
	RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *SLoadbalancerBackendGroup, task taskman.ITask) error
	ValidateDeleteLoadbalancerBackendGroupCondition(ctx context.Context, lbbb *SLoadbalancerBackendGroup) error
	RequestSyncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *SLoadbalancerListener, lbbg *SLoadbalancerBackendGroup, task taskman.ITask) error
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

	ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error)
	ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, backendGroup db.IModel) (*jsonutils.JSONDict, error)
	RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *SLoadbalancerListenerRule, task taskman.ITask) error
	RequestDeleteLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *SLoadbalancerListenerRule, task taskman.ITask) error

	ValidateCreateVpcData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateCreateEipData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)

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
	ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, storage *SStorage, input *api.SSnapshotCreateInput) error
	RequestCreateSnapshot(ctx context.Context, snapshot *SSnapshot, task taskman.ITask) error
	RequestDeleteSnapshot(ctx context.Context, snapshot *SSnapshot, task taskman.ITask) error
	SnapshotIsOutOfChain(disk *SDisk) bool
	GetDiskResetParams(snapshot *SSnapshot) *jsonutils.JSONDict
	OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, snapshot *SSnapshot, data jsonutils.JSONObject) error
	OnSnapshotDelete(ctx context.Context, snapshot *SSnapshot, task taskman.ITask, data jsonutils.JSONObject) error

	//Nat gateway
	DealNatGatewaySpec(spec string) string
	RequestBindIPToNatgateway(ctx context.Context, task taskman.ITask, natgateway *SNatGateway, eipID string) error
	RequestUnBindIPFromNatgateway(ctx context.Context, task taskman.ITask, nat INatHelper, natgateway *SNatGateway) error
	BindIPToNatgatewayRollback(ctx context.Context, eipId string) error

	RequestCacheSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, vpc *SVpc, secgroup *SSecurityGroup, classic bool, task taskman.ITask) error
	RequestSyncSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, vpcId string, vpc *SVpc, secgroup *SSecurityGroup) (string, error)
	IsSupportClassicSecurityGroup() bool
	IsSecurityGroupBelongVpc() bool
	GetDefaultSecurityGroupVpcId() string
	GetSecurityGroupVpcId(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, host *SHost, vpc *SVpc, classic bool) string

	ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.SDBInstanceCreateInput, skus []SDBInstanceSku, network *SNetwork) (*api.SDBInstanceCreateInput, error)
	ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *SDBInstance, input *api.SDBInstanceAccountCreateInput) (*api.SDBInstanceAccountCreateInput, error)
	ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *SDBInstance, input *api.SDBInstanceDatabaseCreateInput) (*api.SDBInstanceDatabaseCreateInput, error)
	ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *SDBInstance, input *api.SDBInstanceBackupCreateInput) (*api.SDBInstanceBackupCreateInput, error)
	ValidateChangeDBInstanceConfigData(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, input *api.SDBInstanceChangeConfigInput) error
	ValidateDBInstanceAccountPrivilege(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, account string, privilege string) error
	ValidateResetDBInstancePassword(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, account string) error

	RequestCreateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *SDBInstance, task taskman.ITask) error
	RequestCreateDBInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, backup *SDBInstanceBackup, task taskman.ITask) error
	IsSupportDBInstancePublicConnection() bool
	IsSupportKeepDBInstanceManualBackup() bool

	IsSupportedBillingCycle(bc billing.SBillingCycle, resource string) bool
	GetSecgroupVpcid(vpcId string) string
	InitDBInstanceUser(dbinstance *SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error
	RequestRenewDBInstance(instance *SDBInstance, bc billing.SBillingCycle) (time.Time, error)
	RequestChangeDBInstanceConfig(ctx context.Context, userCred mcclient.TokenCredential, instance *SDBInstance, task taskman.ITask) error
}

type IElasticcacheDriver interface {
	ValidateCreateElasticcacheData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateCreateElasticcacheAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateCreateElasticcacheAclData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ValidateCreateElasticcacheBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	RequestCreateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestCreateElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheAccount *SElasticcacheAccount, task taskman.ITask) error
	RequestCreateElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheAcl *SElasticcacheAcl, task taskman.ITask) error
	RequestCreateElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, elasticcacheBackup *SElasticcacheBackup, task taskman.ITask) error
	RequestRestartElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestSyncElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestDeleteElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestDeleteElasticcacheAccount(ctx context.Context, userCred mcclient.TokenCredential, ea *SElasticcacheAccount, task taskman.ITask) error
	RequestDeleteElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, ea *SElasticcacheAcl, task taskman.ITask) error
	RequestDeleteElasticcacheBackup(ctx context.Context, userCred mcclient.TokenCredential, eb *SElasticcacheBackup, task taskman.ITask) error
	RequestChangeElasticcacheSpec(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error // 变更实例规格
	RequestSetElasticcacheMaintainTime(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheChangeSpec(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestUpdateElasticcacheAuthMode(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheSetMaintainTime(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheAllocatePublicConnection(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheReleasePublicConnection(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheFlushInstance(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheUpdateInstanceParameters(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
	RequestElasticcacheUpdateBackupPolicy(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, task taskman.ITask) error
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
