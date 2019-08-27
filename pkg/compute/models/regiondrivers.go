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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IRegionDriver interface {
	GetProvider() string

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
	ValidateCreateSnapshotPolicyData(context.Context, mcclient.TokenCredential, *compute.SSnapshotPolicyCreateInput, mcclient.IIdentityProvider, *jsonutils.JSONDict) error
	RequestCreateSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, sp *SSnapshotPolicy, task taskman.ITask) error
	RequestDeleteSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, sp *SSnapshotPolicy, task taskman.ITask) error

	// Region Driver Snapshot Policy joint Disk Apis
	ValidateCreateSnapshopolicyDiskData(ctx context.Context, userCred mcclient.TokenCredential, diskID string) error

	// Region Driver Snapshot Apis
	ValidateSnapshotDelete(ctx context.Context, snapshot *SSnapshot) error
	ValidateSnapshotCreate(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, data *jsonutils.JSONDict) error
	RequestCreateSnapshot(ctx context.Context, snapshot *SSnapshot, task taskman.ITask) error
	RequestDeleteSnapshot(ctx context.Context, snapshot *SSnapshot, task taskman.ITask) error
	SnapshotIsOutOfChain(disk *SDisk) bool
	GetDiskResetParams(snapshot *SSnapshot) *jsonutils.JSONDict
	OnDiskReset(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk, snapshot *SSnapshot, data jsonutils.JSONObject) error
	RequestApplySnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, sp *SSnapshotPolicy, task taskman.ITask, diskId string) error
	RequestCancelSnapshotPolicy(ctx context.Context, userCred mcclient.TokenCredential, sp *SSnapshotPolicy, task taskman.ITask, diskId string) error
	OnSnapshotDelete(ctx context.Context, snapshot *SSnapshot, task taskman.ITask, data jsonutils.JSONObject) error

	//Nat gateway
	DealNatGatewaySpec(spec string) string
	RequestBingToNatgateway(ctx context.Context, task taskman.ITask, natgateway *SNatGateway, needBind bool, eipID string) error
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
