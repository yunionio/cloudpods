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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SRouteTableAssociationManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SRouteTableResourceBaseManager
}

var RouteTableAssociationManager *SRouteTableAssociationManager

func init() {
	RouteTableAssociationManager = &SRouteTableAssociationManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SRouteTableAssociation{},
			"route_table_associations_tbl",
			"route_table_association",
			"route_table_associations",
		),
	}
	RouteTableAssociationManager.SetVirtualObject(RouteTableAssociationManager)
}

type SRouteTableAssociation struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SRouteTableResourceBase
	AssociationType         string `width:"36" charset:"ascii" nullable:"false" list:"user" update:"domain" create:"domain_required"`
	AssociatedResourceId    string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"domain" create:"domain_required"`
	ExtAssociatedResourceId string `width:"36" charset:"ascii" nullable:"false" list:"user" update:"domain" create:"domain_required"`
}

func (manager *SRouteTableAssociationManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{RouteTableManager},
	}
}

func (manager *SRouteTableAssociationManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.RouteTableAssociationListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SRouteTableResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RouteTableFilterList)
	if err != nil {
		return nil, errors.Wrap(err, "SRouteTableResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (self *SRouteTableAssociation) syncRemoveAssociation(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil {
		return err
	}
	err = self.RealDelete(ctx, userCred)
	return err
}

func (self *SRouteTableAssociation) syncWithCloudAssociation(ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	cloudAssociation cloudprovider.RouteTableAssociation,
) error {
	AssociatedResourceId := ""
	if cloudAssociation.AssociationType == cloudprovider.RouteTableAssociaToSubnet {
		routeTable, err := self.GetRouteTable()
		if err != nil {
			return errors.Wrap(err, "self.GetRouteTable()")
		}
		vpc, _ := routeTable.GetVpc()
		subnet, err := vpc.GetNetworkByExtId(cloudAssociation.AssociatedResourceId)
		if err == nil {
			AssociatedResourceId = subnet.GetId()
		}
	}

	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.AssociationType = string(cloudAssociation.AssociationType)
		self.ExtAssociatedResourceId = cloudAssociation.AssociatedResourceId
		self.AssociatedResourceId = AssociatedResourceId
		return nil
	})
	if err != nil {
		return err
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (manager *SRouteTableAssociationManager) newAssociationFromCloud(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	routeTable *SRouteTable,
	provider *SCloudprovider,
	cloudAssociation cloudprovider.RouteTableAssociation,
) (*SRouteTableAssociation, error) {
	association := &SRouteTableAssociation{
		AssociationType:         string(cloudAssociation.AssociationType),
		ExtAssociatedResourceId: cloudAssociation.AssociatedResourceId,
	}
	association.RouteTableId = routeTable.GetId()
	association.ExternalId = cloudAssociation.GetGlobalId()
	if association.AssociationType == string(cloudprovider.RouteTableAssociaToSubnet) {
		vpc, _ := routeTable.GetVpc()
		subnet, err := vpc.GetNetworkByExtId(association.ExtAssociatedResourceId)
		if err == nil {
			association.AssociatedResourceId = subnet.GetId()
		}
	}

	association.SetModelManager(manager, association)
	if err := manager.TableSpec().Insert(ctx, association); err != nil {
		return nil, err
	}

	db.OpsLog.LogEvent(association, db.ACT_CREATE, association.GetShortDesc(ctx), userCred)
	return association, nil
}

func (self *SRouteTableAssociation) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SRouteTableAssociation) GetRouteTable() (*SRouteTable, error) {
	routeTable, err := RouteTableManager.FetchById(self.RouteTableId)
	if err != nil {
		return nil, errors.Wrapf(err, "RouteTableManager.FetchById(%s)", self.RouteTableId)
	}
	return routeTable.(*SRouteTable), nil
}

func (manager *SRouteTableAssociationManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}

	q, err = manager.SRouteTableResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SRouteTableResourceBaseManager.ListItemExportKeys")
	}

	return q, nil
}
