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

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SBaseRegionDriver struct {
}

func (self *SBaseRegionDriver) RequestCreateLoadbalancer(ctx context.Context, userCred mcclient.TokenCredential, lb *models.SLoadbalancer, task taskman.ITask) error {
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

func (self *SBaseRegionDriver) ValidateCreateSnapshotData(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, storage *models.SStorage, input *api.SSnapshotCreateInput) error {
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

func (self *SBaseRegionDriver) DealNatGatewaySpec(spec string) string {
	return spec
}

func (self *SBaseRegionDriver) RequestBingToNatgateway(ctx context.Context, task taskman.ITask,
	natgateway *models.SNatGateway, needBind bool, eipID string) error {

	return fmt.Errorf("Not implement RequestBindIPToNatgateway")
}

func (self *SBaseRegionDriver) RequestCacheSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, secgroup *models.SSecurityGroup, classic bool, task taskman.ITask) error {
	return fmt.Errorf("Not Implemented RequestCacheSecurityGroup")
}

func (self *SBaseRegionDriver) IsSupportClassicSecurityGroup() bool {
	return false
}

func (self *SBaseRegionDriver) IsSecurityGroupBelongVpc() bool {
	return false
}

func (self *SBaseRegionDriver) GetDefaultSecurityGroupVpcId() string {
	return api.NORMAL_VPC_ID
}

func (self *SBaseRegionDriver) GetSecurityGroupVpcId(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, host *models.SHost, vpc *models.SVpc, classic bool) string {
	return ""
}

func (self *SBaseRegionDriver) RequestSyncSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, vpcId string, vpc *models.SVpc, secgroup *models.SSecurityGroup) (string, error) {
	return "", fmt.Errorf("Not Implemented RequestSyncSecurityGroup")
}

func (self *SBaseRegionDriver) ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.SDBInstanceCreateInput, skus []models.SDBInstanceSku, network *models.SNetwork) (*api.SDBInstanceCreateInput, error) {
	return input, nil
}

func (self *SBaseRegionDriver) RequestCreateDBInstance(ctx context.Context, userCred mcclient.TokenCredential, dbinstance *models.SDBInstance, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestCreateDBInstance")
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

func (self *SBaseRegionDriver) InitDBInstanceUser(dbinstance *models.SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error {
	return nil
}

func (self *SBaseRegionDriver) RequestRenewDBInstance(instance *models.SDBInstance, bc billing.SBillingCycle) (time.Time, error) {
	return time.Time{}, fmt.Errorf("Not Implement RequestRenewDBInstance")
}

func (self *SBaseRegionDriver) RequestChangeDBInstanceConfig(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, task taskman.ITask) error {
	return fmt.Errorf("Not Implement RequestChangeDBInstanceConfig")
}

func (self *SBaseRegionDriver) ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input *api.SDBInstanceAccountCreateInput) (*api.SDBInstanceAccountCreateInput, error) {
	return nil, fmt.Errorf("Not Implement ValidateCreateDBInstanceAccountData")
}

func (self *SBaseRegionDriver) ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input *api.SDBInstanceDatabaseCreateInput) (*api.SDBInstanceDatabaseCreateInput, error) {
	return nil, fmt.Errorf("Not Implement ValidateCreateDBInstanceDatabaseData")
}

func (self *SBaseRegionDriver) ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input *api.SDBInstanceBackupCreateInput) (*api.SDBInstanceBackupCreateInput, error) {
	return nil, fmt.Errorf("Not Implement ValidateCreateDBInstanceBackupData")
}

func (self *SBaseRegionDriver) ValidateChangeDBInstanceConfigData(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, input *api.SDBInstanceChangeConfigInput) error {
	return fmt.Errorf("Not Implement ValidateChangeDBInstanceConfigData")
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
