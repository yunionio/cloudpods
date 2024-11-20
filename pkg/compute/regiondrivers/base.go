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
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (self *SBaseRegionDriver) RequestCreateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerAcl")
}

func (self *SBaseRegionDriver) RequestUpdateLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestUpdateLoadbalancerAcl")
}

func (self *SBaseRegionDriver) RequestLoadbalancerAclSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestLoadbalancerAclSyncstatus")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, lbacl *models.SLoadbalancerAcl, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerAcl")
}

func (self *SBaseRegionDriver) IsCertificateBelongToRegion() bool {
	return true
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerCertificate")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerCertificate(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerCertificate")
}

func (self *SBaseRegionDriver) RequestLoadbalancerCertificateSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, lbcert *models.SLoadbalancerCertificate, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestLoadbalancerCertificateSyncstatus")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateLoadbalancerBackendGroup")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerBackendGroup(ctx context.Context, userCred mcclient.TokenCredential, lbbg *models.SLoadbalancerBackendGroup, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerBackendGroup")
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

func (self *SBaseRegionDriver) ValidateCreateLoadbalancerListenerData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerCreateInput,
	lb *models.SLoadbalancer, lbbg *models.SLoadbalancerBackendGroup) (*api.LoadbalancerListenerCreateInput, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateLoadbalancerListenerData")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateLoadbalancerListener")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestDeleteLoadbalancerListener")
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

func (self *SBaseRegionDriver) RequestSyncLoadbalancerListener(ctx context.Context, userCred mcclient.TokenCredential, lblis *models.SLoadbalancerListener, input *api.LoadbalancerListenerUpdateInput, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestSyncLoadbalancerListener")
}

func (self *SBaseRegionDriver) ValidateCreateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LoadbalancerListenerRuleCreateInput) (*api.LoadbalancerListenerRuleCreateInput, error) {
	return input, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateLoadbalancerListenerRuleData")
}

func (self *SBaseRegionDriver) ValidateUpdateLoadbalancerListenerRuleData(ctx context.Context, userCred mcclient.TokenCredential, input *api.LoadbalancerListenerRuleUpdateInput) (*api.LoadbalancerListenerRuleUpdateInput, error) {
	return input, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateUpdateLoadbalancerListenerRuleData")
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateLoadbalancerListenerRule")
}

func (self *SBaseRegionDriver) RequestDeleteLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential, lbr *models.SLoadbalancerListenerRule, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestDeleteLoadbalancerListenerRule")
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

func (self *SBaseRegionDriver) ValidateCreateSecurityGroupInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupCreateInput) (*api.SSecgroupCreateInput, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateSecurityGroupInput")
}

func (self *SBaseRegionDriver) GetDefaultSecurityGroupNamePrefix() string {
	return "default-auto"
}

func (self *SBaseRegionDriver) RequestCreateSecurityGroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	secgroup *models.SSecurityGroup,
	rules api.SSecgroupRuleResourceSet,
) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateSecurityGroup")
}

func (self *SBaseRegionDriver) RequestDeleteSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, secgroup *models.SSecurityGroup, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestDeleteSecurityGroup")
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

func (self *SBaseRegionDriver) ValidateCreateWafInstanceData(ctx context.Context, userCred mcclient.TokenCredential, input api.WafInstanceCreateInput) (api.WafInstanceCreateInput, error) {
	return input, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateWafInstanceData")
}

func (self *SBaseRegionDriver) ValidateCreateWafRuleData(ctx context.Context, userCred mcclient.TokenCredential, waf *models.SWafInstance, input api.WafRuleCreateInput) (api.WafRuleCreateInput, error) {
	return input, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateWafRuleData")
}

func (self *SBaseRegionDriver) RequestCreateNetwork(ctx context.Context, userCred mcclient.TokenCredential, net *models.SNetwork, task taskman.ITask) error {
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

func (self *SBaseRegionDriver) RequestRemoteUpdateElasticSearch(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *models.SElasticSearch, replaceTags bool, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestRemoteUpdateElasticSearch")
}

func (self *SBaseRegionDriver) RequestRemoteUpdateKafka(ctx context.Context, userCred mcclient.TokenCredential, kafka *models.SKafka, replaceTags bool, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestRemoteUpdateElasticSearch")
}

func (self *SBaseRegionDriver) ValidateCreateKubeClusterData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.KubeClusterCreateInput) (*api.KubeClusterCreateInput, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateKubeClusterData")
}

func (self *SBaseRegionDriver) ValidateCreateKubeNodePoolData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.KubeNodePoolCreateInput) (*api.KubeNodePoolCreateInput, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateKubeNodePoolData")
}

func (self *SBaseRegionDriver) RequestCreateKubeCluster(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SKubeCluster, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateKubeCluster")
}

func (self *SBaseRegionDriver) RequestCreateKubeNodePool(ctx context.Context, userCred mcclient.TokenCredential, pool *models.SKubeNodePool, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateKubeNodePool")
}

func (drv *SBaseRegionDriver) RequestPrepareSecurityGroups(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	secgroups []models.SSecurityGroup,
	vpc *models.SVpc,
	callback func(ids []string) error,
	task taskman.ITask,
) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestPrepareSecurityGroups")
}

func (drv *SBaseRegionDriver) CreateDefaultSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, vpc *models.SVpc) (*models.SSecurityGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateDefaultSecurityGroup")
}

func (drv *SBaseRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetSecurityGroupFilter")
}

func (self *SBaseRegionDriver) ValidateUpdateSecurityGroupRuleInput(ctx context.Context, userCred mcclient.TokenCredential, input *api.SSecgroupRuleUpdateInput) (*api.SSecgroupRuleUpdateInput, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateUpdateSecurityGroupInput")
}

func (self *SBaseRegionDriver) ValidateCreateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, input *api.SSnapshotPolicyCreateInput) (*api.SSnapshotPolicyCreateInput, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateSnapshotPolicy")
}

func (self *SBaseRegionDriver) RequestCreateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, sp *models.SSnapshotPolicy, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateSnapshotPolicy")
}

func (self *SBaseRegionDriver) RequestDeleteSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, sp *models.SSnapshotPolicy, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestDeleteSnapshotPolicy")
}

func (self *SBaseRegionDriver) RequestSnapshotPolicyBindDisks(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, diskIds []string, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestSnapshotPolicyBindDisks")
}

func (self *SBaseRegionDriver) RequestSnapshotPolicyUnbindDisks(ctx context.Context, userCred mcclient.TokenCredential, sp *models.SSnapshotPolicy, diskIds []string, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestSnapshotPolicyUnbindDisks")
}

func (self *SBaseRegionDriver) RequestRemoteUpdateNetwork(ctx context.Context, userCred mcclient.TokenCredential, network *models.SNetwork, replaceTags bool, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestRemoteUpdateNetwork")
}
