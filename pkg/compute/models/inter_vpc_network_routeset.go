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
	"database/sql"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SInterVpcNetworkRouteSetManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SVpcResourceBaseManager
}

var InterVpcNetworkRouteSetManager *SInterVpcNetworkRouteSetManager

func init() {
	InterVpcNetworkRouteSetManager = &SInterVpcNetworkRouteSetManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SInterVpcNetworkRouteSet{},
			"inter_vpc_network_route_sets_tbl",
			"inter_vpc_network_route_set",
			"inter_vpc_network_route_sets",
		),
	}
	InterVpcNetworkRouteSetManager.SetVirtualObject(InterVpcNetworkRouteSetManager)
}

type SInterVpcNetworkRouteSet struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SVpcResourceBase
	InterVpcNetworkId string

	Cidr                string `width:"36" charset:"ascii" nullable:"true" list:"domain"`
	ExtInstanceId       string `width:"36" charset:"ascii" nullable:"false" list:"domain"`
	ExtInstanceType     string `width:"36" charset:"ascii" nullable:"false" list:"domain"`
	ExtInstanceRegionId string `width:"36" charset:"ascii" nullable:"false" list:"domain"`
}

func (manager *SInterVpcNetworkRouteSetManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.InterVpcNetworkRouteSetListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SInterVpcNetworkRouteSetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.InterVpcNetworkRouteSetListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	if len(query.InterVpcNetworkId) > 0 {
		vpcNetwork, err := InterVpcNetworkManager.FetchByIdOrName(ctx, userCred, query.InterVpcNetworkId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("inter_vpc_network_id", query.InterVpcNetworkId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("inter_vpc_network_id", vpcNetwork.GetId())
	}
	if len(query.Cidr) > 0 {
		q = q.Equals("cidr", query.Cidr)
	}
	return q, nil
}

func (self *SInterVpcNetworkRouteSet) syncRemoveRouteSet(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	err = self.RealDelete(ctx, userCred)
	return err
}

func (self *SInterVpcNetworkRouteSet) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SInterVpcNetworkRouteSet) syncWithCloudRouteSet(ctx context.Context, userCred mcclient.TokenCredential, interVpcNetwork *SInterVpcNetwork, cloudRouteSet cloudprovider.ICloudInterVpcNetworkRoute) error {
	vpcId := ""
	if cloudRouteSet.GetInstanceType() == api.INTER_VPCNETWORK_ATTACHED_INSTAMCE_TYPE_VPC {
		provider := interVpcNetwork.GetCloudprovider()
		vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, cloudRouteSet.GetInstanceId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			managerQ := CloudproviderManager.Query("id").Equals("provider", provider.Provider)
			return q.In("manager_id", managerQ.SubQuery())
		})
		if err != nil {
			return errors.Wrap(err, "db.FetchByExternalIdAndManagerId(VpcManager")
		}
		vpcId = vpc.GetId()
	}

	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Name = cloudRouteSet.GetName()
		self.Enabled = tristate.NewFromBool(cloudRouteSet.GetEnabled())
		self.Status = cloudRouteSet.GetStatus()
		self.Cidr = cloudRouteSet.GetCidr()
		self.InterVpcNetworkId = interVpcNetwork.GetId()
		self.VpcId = vpcId
		self.ExtInstanceId = cloudRouteSet.GetInstanceId()
		self.ExtInstanceType = cloudRouteSet.GetInstanceType()
		self.ExtInstanceRegionId = cloudRouteSet.GetInstanceRegionId()
		return nil
	})
	if err != nil {
		return err
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SInterVpcNetworkRouteSetManager) newRouteSetFromCloud(ctx context.Context, userCred mcclient.TokenCredential, interVpcNetwork *SInterVpcNetwork, cloudRouteSet cloudprovider.ICloudInterVpcNetworkRoute) (*SInterVpcNetworkRouteSet, error) {
	routeSet := &SInterVpcNetworkRouteSet{
		InterVpcNetworkId:   interVpcNetwork.GetId(),
		Cidr:                cloudRouteSet.GetCidr(),
		ExtInstanceId:       cloudRouteSet.GetInstanceId(),
		ExtInstanceType:     cloudRouteSet.GetInstanceType(),
		ExtInstanceRegionId: cloudRouteSet.GetInstanceRegionId(),
	}

	if cloudRouteSet.GetInstanceType() == api.INTER_VPCNETWORK_ATTACHED_INSTAMCE_TYPE_VPC {
		provider := interVpcNetwork.GetCloudprovider()
		vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, cloudRouteSet.GetInstanceId(), func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			managerQ := CloudproviderManager.Query("id").Equals("provider", provider.Provider)
			return q.In("manager_id", managerQ.SubQuery())
		})
		if err != nil {
			return nil, errors.Wrap(err, "db.FetchByExternalIdAndManagerId(VpcManager")
		}
		routeSet.VpcId = vpc.GetId()
	}
	routeSet.ExternalId = cloudRouteSet.GetId()
	routeSet.Name = cloudRouteSet.GetName()
	routeSet.Enabled = tristate.NewFromBool(cloudRouteSet.GetEnabled())
	routeSet.Status = cloudRouteSet.GetStatus()

	routeSet.SetModelManager(manager, routeSet)
	if err := manager.TableSpec().Insert(ctx, routeSet); err != nil {
		return nil, err
	}

	db.OpsLog.LogEvent(routeSet, db.ACT_CREATE, routeSet.GetShortDesc(ctx), userCred)
	return routeSet, nil
}

func (manager *SInterVpcNetworkRouteSetManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}

	q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
	}

	return q, nil
}

func (manager *SInterVpcNetworkRouteSetManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SInterVpcNetworkRouteSetManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.InterVpcNetworkRouteSetDetails {
	rows := make([]api.InterVpcNetworkRouteSetDetails, len(objs))
	stdRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.InterVpcNetworkRouteSetDetails{
			EnabledStatusStandaloneResourceDetails: stdRows[i],
			VpcResourceInfo:                        vpcRows[i],
		}
	}
	return rows
}

func (self *SInterVpcNetworkRouteSet) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.InterVpcNetworkRouteSetEnableInput) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, input.PerformEnableInput)
	if err != nil {
		return nil, err
	}
	network, err := self.GetInterVpcNetwork()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetInterVpcNetwork()")
	}
	err = network.StartInterVpcNetworkUpdateRoutesetTask(ctx, userCred, self, "enable")

	return nil, err
}

func (self *SInterVpcNetworkRouteSet) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.InterVpcNetworkRouteSetDisableInput) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, input.PerformDisableInput)
	if err != nil {
		return nil, err
	}
	network, err := self.GetInterVpcNetwork()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetInterVpcNetwork()")
	}
	err = network.StartInterVpcNetworkUpdateRoutesetTask(ctx, userCred, self, "disable")
	return nil, err
}

func (self *SInterVpcNetworkRouteSet) GetInterVpcNetwork() (*SInterVpcNetwork, error) {
	network, err := InterVpcNetworkManager.FetchById(self.InterVpcNetworkId)
	if err != nil {
		return nil, errors.Wrapf(err, "InterVpcNetworkManager.FetchById(%s)", self.InterVpcNetworkId)
	}
	return network.(*SInterVpcNetwork), nil
}

func (self *SInterVpcNetwork) StartInterVpcNetworkUpdateRoutesetTask(ctx context.Context, userCred mcclient.TokenCredential, routeSet *SInterVpcNetworkRouteSet, routeSetAction string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(routeSetAction), "action")
	params.Add(jsonutils.NewString(routeSet.GetId()), "inter_vpc_network_route_set_id")
	task, err := taskman.TaskManager.NewTask(ctx, "InterVpcNetworkUpdateRoutesetTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "Start InterVpcNetworkUpdateRoutesetTask fail")
	}
	self.SetStatus(ctx, userCred, api.INTER_VPC_NETWORK_STATUS_UPDATEROUTE, "update route")
	task.ScheduleRun(nil)
	return nil
}
