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

package regiondrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/pinyinutils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SBaseRegionDriver struct {
}

func (self *SBaseRegionDriver) ValidateCreateLoadbalancerData(ctx context.Context, userCred mcclient.TokenCredential, owerId mcclient.IIdentityProvider, input *api.LoadbalancerCreateInput) (*api.LoadbalancerCreateInput, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateLoadbalancerData")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerInstance(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, input *api.LoadbalancerCreateInput, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancer")
}

func (self *SBaseRegionDriver) RequestStartLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestStartLoadbalancer")
}

func (self *SBaseRegionDriver) RequestStopLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestStopLoadbalancer")
}

func (self *SBaseRegionDriver) RequestSyncstatusLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncstatusLoadbalancer")
}

func (self *SBaseRegionDriver) RequestRemoteUpdateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, replaceTags bool, task taskman.ITask) error {
	// nil ops
	return nil
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancer")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerAcl")
}

func (self *SBaseRegionDriver) RequestSyncLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncLoadbalancerAcl")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SCachedLoadbalancerAcl, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerAcl")
}

func (self *SBaseRegionDriver) IsCertificateBelongToRegion() bool {
	return true
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SCachedLoadbalancerCertificate, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerCertificate")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SCachedLoadbalancerCertificate, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerCertificate")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, backends []cloudprovider.SLoadbalancerBackend, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerBackendGroup")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerBackendGroup")
}

func (self *SBaseRegionDriver) RequestSyncLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncLoadbalancerBackendGroup")
}

func (self *SBaseRegionDriver) RequestPullRegionLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults models.SSyncResultSet, provider *models.SCloudprovider, localRegion *models.SCloudregion, remoteRegion cloudprovider.ICloudRegion, syncRange *models.SSyncRange) error {
	return fmt.Errorf("Not Implement RequestPullRegionLoadbalancerBackendGroup")
}

func (self *SBaseRegionDriver) RequestPullLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, syncResults models.SSyncResultSet, provider *models.SCloudprovider, localLoadbalancer *models.SLoadbalancer, remoteLoadbalancer cloudprovider.ICloudLoadbalancer, syncRange *models.SSyncRange) error {
	return fmt.Errorf("Not Implement RequestPullLoadbalancerBackendGroup")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerBackend")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerBackend")
}

func (self *SBaseRegionDriver) RequestSyncLoadbalancerBackend(ctx context.Context, userCred mcclient.TokenCredential, lbb *models.SLoadbalancerBackend, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncLoadbalancerBackend")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerListener")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerListener")
}

func (self *SBaseRegionDriver) RequestStartLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestStartLoadbalancerListener")
}

func (self *SBaseRegionDriver) RequestStopLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestStopLoadbalancerListener")
}

func (self *SBaseRegionDriver) RequestSyncstatusLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncstatusLoadbalancerListener")
}

func (self *SBaseRegionDriver) RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncLoadbalancerListener")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerListenerRule")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerListenerRule")
}

func (self *SBaseRegionDriver) RequestUpdateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, input cloudprovider.SnapshotPolicyInput, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestUpdateSnapshotPolicy")
}

func (self *SBaseRegionDriver) RequestApplySnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, task taskman.ITask, disk *models.SDisk, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) error {
	return fmt.Errorf("Not Implement RequestApplySnapshotPolicy")
}

func (self *SBaseRegionDriver) RequestCancelSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, task taskman.ITask, disk *models.SDisk, sp *models.SSnapshotPolicy, data jsonutils.JSONObject) error {
	return fmt.Errorf("Not Implement RequestCancelSnapshotPolicy")
}

func (self *SBaseRegionDriver) ValidateSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot) error {
	return fmt.Errorf("Not Implement ValidateSnapshotDelete")
}

func (self *SBaseRegionDriver) RequestDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteSnapshot")
}

func (self *SBaseRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SnapshotCreateInput) error {
	return fmt.Errorf("Not Implement ValidateCreateSnapshotData")
}

func (self *SBaseRegionDriver) RequestCreateSnapshot(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateSnapshot")
}

func (self *SBaseRegionDriver) SnapshotIsOutOfChain(disk *models.SDisk) bool {
	return true
}

func (self *SBaseRegionDriver) GetDiskResetParams(snapshot *models.SSnapshot) *jsonutils.JSONDict {
	return nil
}

func (self *SBaseRegionDriver) OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, data *jsonutils.JSONObject) error {
	return fmt.Errorf("Not Implement OnDiskReset")
}

func (self *SBaseRegionDriver) ValidateCreateSnapshopolicyDiskData(ctx context.Context,
	userCred mcclient.TokenCredential, disk *models.SDisk, snapshotPolicy *models.SSnapshotPolicy) error {

	if disk.DomainId != snapshotPolicy.DomainId {
		return httperrors.NewBadRequestError("disk and snapshotpolicy should have same domain")
	}
	if disk.ProjectId != snapshotPolicy.ProjectId {
		return httperrors.NewBadRequestError("disk and snapshotpolicy should have same project")
	}
	return nil
}

func (self *SBaseRegionDriver) OnSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot, task taskman.ITask, data jsonutils.JSONObject) error {
	return fmt.Errorf("Not implement OnSnapshotDelete")
}

func (self *SBaseRegionDriver) RequestAssociateEipForNAT(ctx context.Context, userCred mcclient.TokenCredential, nat *models.SNatGateway, eip *models.SElasticip, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestAssociateEipForNAT")
}

func (self *SBaseRegionDriver) IsVpcCreateNeedInputCidr() bool {
	return true
}

func (self *SBaseRegionDriver) RequestCreateVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	return fmt.Errorf("Not implement RequestCreateVpc")
}

func (self *SBaseRegionDriver) RequestDeleteVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	return fmt.Errorf("Not implement RequestDeleteVpc")
}

func (self *SBaseRegionDriver) IsAllowSecurityGroupNameRepeat() bool {
	return false
}

func (self *SBaseRegionDriver) GenerateSecurityGroupName(name string) string {
	return pinyinutils.Text2Pinyin(name)
}

func (self *SBaseRegionDriver) RequestCacheSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, secgroup *models.SSecurityGroup, classic bool, remoteProjectId string, task taskman.ITask) error {
	return fmt.Errorf("Not Implemented RequestCacheSecurityGroup")
}

func (self *SBaseRegionDriver) IsSupportClassicSecurityGroup() bool {
	return false
}

func (self *SBaseRegionDriver) IsSecurityGroupBelongVpc() bool {
	return false
}

func (self *SBaseRegionDriver) IsVpcBelongGlobalVpc() bool {
	return false
}

func (self *SBaseRegionDriver) IsSecurityGroupBelongGlobalVpc() bool {
	return false
}

func (self *SBaseRegionDriver) GetDefaultSecurityGroupVpcId() string {
	return api.NORMAL_VPC_ID
}

func (self *SBaseRegionDriver) GetSecurityGroupPublicScope(service string) rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (self *SBaseRegionDriver) GetSecurityGroupVpcId(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, host *models.SHost, vpc *models.SVpc, classic bool) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SBaseRegionDriver) RequestSyncSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, vpcId string, vpc *models.SVpc, secgroup *models.SSecurityGroup, removeProjectId, service string, skipSyncRule bool) (string, error) {
	return "", fmt.Errorf("Not Implemented RequestSyncSecurityGroup")
}

func (self *SBaseRegionDriver) IsOnlySupportAllowRules() bool {
	return false
}

func (self *SBaseRegionDriver) IsSupportPeerSecgroup() bool {
	return false
}

func (self *SBaseRegionDriver) IsPeerSecgroupWithSameProject() bool {
	return false
}

func (self *SBaseRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SBaseRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:allow any")}
}

func (self *SBaseRegionDriver) GetSecurityGroupRuleMaxPriority() int {
	return 100
}

func (self *SBaseRegionDriver) GetSecurityGroupRuleMinPriority() int {
	return 1
}

func (self *SBaseRegionDriver) ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.DBInstanceCreateInput, skus []models.SDBInstanceSku, network *models.SNetwork) (api.DBInstanceCreateInput, error) {
	return input, nil
}

func (self *SBaseRegionDriver) RequestCreateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *models.SDBInstance, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateDBInstance")
}

func (self *SBaseRegionDriver) RequestCreateDBInstanceFromBackup(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *models.SDBInstance, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateDBInstanceFromBackup")
}

func (self *SBaseRegionDriver) RequestCreateDBInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *models.SDBInstance, backup *models.SDBInstanceBackup, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateDBInstanceBackup")
}

func (self *SBaseRegionDriver) IsSupportedBillingCycle(bc billing.SBillingCycle, resource string) bool {
	return false
}

func (self *SBaseRegionDriver) GetSecgroupVpcid(vpcId string) string {
	return vpcId
}

func (self *SBaseRegionDriver) InitDBInstanceUser(ctx context.Context, dbinstance *models.SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error {
	return nil
}

func (self *SBaseRegionDriver) RequestChangeDBInstanceConfig(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, input *api.SDBInstanceChangeConfigInput, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestChangeDBInstanceConfig")
}

func (self *SBaseRegionDriver) ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceAccountCreateInput) (api.DBInstanceAccountCreateInput, error) {
	return input, fmt.Errorf("Not Implement ValidateCreateDBInstanceAccountData")
}

func (self *SBaseRegionDriver) ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceDatabaseCreateInput) (api.DBInstanceDatabaseCreateInput, error) {
	return input, fmt.Errorf("Not Implement ValidateCreateDBInstanceDatabaseData")
}

func (self *SBaseRegionDriver) ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceBackupCreateInput) (api.DBInstanceBackupCreateInput, error) {
	return input, fmt.Errorf("Not Implement ValidateCreateDBInstanceBackupData")
}

func (self *SBaseRegionDriver) ValidateDBInstanceAccountPrivilege(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, account string, privilege string) error {
	return fmt.Errorf("Not Implement ValidateDBInstanceAccountPrivilege")
}

func (self *SBaseRegionDriver) IsSupportDBInstancePublicConnection() bool {
	return true
}

func (self *SBaseRegionDriver) ValidateResetDBInstancePassword(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, account string) error {
	return nil
}

func (self *SBaseRegionDriver) IsSupportKeepDBInstanceManualBackup() bool {
	return false
}

func (self *SBaseRegionDriver) ValidateDBInstanceRecovery(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, backup *models.SDBInstanceBackup, input api.SDBInstanceRecoveryConfigInput) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "ValidateDBInstanceRecovery")
}

func (self *SBaseRegionDriver) RequestRemoteUpdateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, replaceTags bool, task taskman.ITask) error {
	// nil ops
	return nil
}

func (self *SBaseRegionDriver) IsSupportedDBInstance() bool {
	return false
}

func (self *SBaseRegionDriver) IsSupportedDBInstanceAutoRenew() bool {
	return false
}

func (self *SBaseRegionDriver) IsSupportedElasticcache() bool {
	return false
}

func (self *SBaseRegionDriver) RequestSyncDiskStatus(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncDiskStatus")
}

func (self *SBaseRegionDriver) RequestSyncSnapshotStatus(ctx context.Context, userCred mcclient.TokenCredential, snapshot *models.SSnapshot, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncSnapshotStatus")
}

func (self *SBaseRegionDriver) RequestSyncNatGatewayStatus(ctx context.Context, userCred mcclient.TokenCredential, natgateway *models.SNatGateway, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncNatGatewayStatus")
}

func (self *SBaseRegionDriver) RequestSyncBucketStatus(ctx context.Context, userCred mcclient.TokenCredential, bucket *models.SBucket, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncBucketStatus")
}

func (self *SBaseRegionDriver) RequestSyncDBInstanceBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, backup *models.SDBInstanceBackup, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncDBInstanceBackupStatus")
}

func (self *SBaseRegionDriver) RequestCreateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask, data *jsonutils.JSONDict) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateElasticcache")
}

func (self *SBaseRegionDriver) RequestSyncElasticcacheStatus(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncElasticcacheStatus")
}

func (self *SBaseRegionDriver) RequestRemoteUpdateElasticcache(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticcache, replaceTags bool, task taskman.ITask) error {
	// nil ops
	return nil
}

func (self *SBaseRegionDriver) RequestSyncSecgroupsForElasticcache(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestSyncSecgroupsForElasticcache")
}

func (self *SBaseRegionDriver) IsDBInstanceNeedSecgroup() bool {
	return false
}

func (self *SBaseRegionDriver) GetRdsSupportSecgroupCount() int {
	return 0
}

func (self *SBaseRegionDriver) RequestRenewElasticcache(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SElasticcache, bc billing.SBillingCycle) (time.Time, error) {
	return time.Time{}, fmt.Errorf("Not Implement RequestRenewElasticcache")
}

func (self *SBaseRegionDriver) IsSupportedElasticcacheAutoRenew() bool {
	return false
}

func (self *SBaseRegionDriver) RequestElasticcacheSetAutoRenew(ctx context.Context, userCred mcclient.TokenCredential, ec *models.SElasticcache, autoRenew bool, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestElasticcacheSetAutoRenew")
}

func (self *SBaseRegionDriver) RequestSyncRdsSecurityGroups(ctx context.Context, userCred mcclient.TokenCredential, rds *models.SDBInstance, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestSyncRdsSecurityGroups")
}

func (self *SBaseRegionDriver) IsSupportedNatGateway() bool {
	return false
}

func (self *SBaseRegionDriver) IsSupportedNas() bool {
	return false
}

func (self *SBaseRegionDriver) OnNatEntryDeleteComplete(ctx context.Context, userCred mcclient.TokenCredential, eip *models.SElasticip) error {
	return nil
}

func (self *SBaseRegionDriver) ValidateCreateNatGateway(ctx context.Context, userCred mcclient.TokenCredential, input api.NatgatewayCreateInput) (api.NatgatewayCreateInput, error) {
	return input, httperrors.NewNotImplementedError("ValidateCreateNatGateway")
}

func (self *SBaseRegionDriver) IsSupportedNatAutoRenew() bool {
	return true
}

func (self *SBaseRegionDriver) RequestAssociateEip(ctx context.Context, userCred mcclient.TokenCredential, eip *models.SElasticip, input api.ElasticipAssociateInput, obj db.IStatusStandaloneModel, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("RequestAssociateEip")
}

func (self *SBaseRegionDriver) RequestSyncAccessGroup(ctx context.Context, userCred mcclient.TokenCredential, fs *models.SFileSystem, mt *models.SMountTarget, ag *models.SAccessGroup, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestSyncAccessGroup")
}

func (self *SBaseRegionDriver) ValidateCreateWafInstanceData(ctx context.Context, userCred mcclient.TokenCredential, input api.WafInstanceCreateInput) (api.WafInstanceCreateInput, error) {
	return input, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateWafInstanceData")
}

func (self *SBaseRegionDriver) ValidateCreateWafRuleData(ctx context.Context, userCred mcclient.TokenCredential, waf *models.SWafInstance, input api.WafRuleCreateInput) (api.WafRuleCreateInput, error) {
	return input, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateWafRuleData")
}

func (self *SBaseRegionDriver) RequestCreateNetwork(ctx context.Context, userCred mcclient.TokenCredential, net *models.SNetwork) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateNetwork")
}

func (self *SBaseRegionDriver) RequestCreateBackup(ctx context.Context, backup *models.SDiskBackup, snapshotId string, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateBackup")
}

func (self *SBaseRegionDriver) RequestDeleteBackup(ctx context.Context, backup *models.SDiskBackup, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestDeleteBackup")
}

func (self *SBaseRegionDriver) RequestCreateInstanceBackup(ctx context.Context, guest *models.SGuest, ib *models.SInstanceBackup, task taskman.ITask, params *jsonutils.JSONDict) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateInstanceBackup")
}

func (self *SBaseRegionDriver) RequestDeleteInstanceBackup(ctx context.Context, ib *models.SInstanceBackup, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestDeleteInstanceBackup")
}

func (self *SBaseRegionDriver) ValidateCreateCdnData(ctx context.Context, userCred mcclient.TokenCredential, input api.CDNDomainCreateInput) (api.CDNDomainCreateInput, error) {
	return input, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateCdnData")
}

func (self *SBaseRegionDriver) RequestSyncInstanceBackupStatus(ctx context.Context, userCred mcclient.TokenCredential, ib *models.SInstanceBackup, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "SyncInstanceBackupStatus")
}

func (self *SBaseRegionDriver) RequestSyncBackupStorageStatus(ctx context.Context, userCred mcclient.TokenCredential, bs *models.SBackupStorage, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "SyncBackupStorageStatus")
}

func (self *SBaseRegionDriver) RequestPackInstanceBackup(ctx context.Context, ib *models.SInstanceBackup, task taskman.ITask, packageName string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestPackInstanceBackup")
}
func (self *SBaseRegionDriver) RequestUnpackInstanceBackup(ctx context.Context, ib *models.SInstanceBackup, task taskman.ITask, packageName string, metadataOnly bool) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestUnpackInstanceBackup")
}
