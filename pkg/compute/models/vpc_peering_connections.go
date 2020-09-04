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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

type SVpcPeeringConnectionManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
}

var VpcPeeringConnectionManager *SVpcPeeringConnectionManager

func init() {
	VpcPeeringConnectionManager = &SVpcPeeringConnectionManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SVpcPeeringConnection{},
			"vpc_peering_connections_tbl",
			"vpc_peering_connection",
			"vpc_peering_connections",
		),
	}
	VpcPeeringConnectionManager.SetVirtualObject(VpcPeeringConnectionManager)
}

type SVpcPeeringConnection struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SVpcResourceBase
	PeerVpcId string `width:"36" charset:"ascii" nullable:"true" list:"domain" create:"required" json:"peer_vpc_id"`
}

func (manager *SVpcPeeringConnectionManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{VpcManager},
	}
}

// 列表
func (manager *SVpcPeeringConnectionManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.VpcPeeringConnectionListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

// 创建
func (manager *SVpcPeeringConnectionManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.VpcPeeringConnectionCreateInput,
) (api.VpcPeeringConnectionCreateInput, error) {
	var err error
	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SVpcPeeringConnection) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.VpcPeeringConnectionDetails, error) {
	return api.VpcPeeringConnectionDetails{}, nil
}

func (manager *SVpcPeeringConnectionManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.VpcPeeringConnectionDetails {
	rows := make([]api.VpcPeeringConnectionDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.VpcPeeringConnectionDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
		}
	}
	return rows
}

func (self *SVpcPeeringConnection) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SVpcPeeringConnection) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (manager *SVpcPeeringConnectionManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.VpcPeeringConnectionListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.OrderByExtraFields")
	}
	return q, httperrors.ErrNotFound
}

func (manager *SVpcPeeringConnectionManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (self *SVpcPeeringConnection) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.VpcPeeringConnectionUpdateInput) (api.VpcPeeringConnectionUpdateInput, error) {
	var err error
	input.EnabledStatusInfrasResourceBaseUpdateInput, err = self.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SVpcPeeringConnection) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SVpcPeeringConnection) SyncWithCloudPeerConnection(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudVpcPeeringConnection, provider *SCloudprovider) error {
	_, err := db.Update(self, func() error {
		self.Status = ext.GetStatus()
		self.ExternalId = ext.GetGlobalId()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	if provider != nil {
		SyncCloudDomain(userCred, self, provider.GetOwnerId())
		self.SyncShareState(ctx, userCred, provider.getAccountShareInfo())
	}
	return nil
}

func (self *SVpcPeeringConnection) GetVpc() (*SVpc, error) {
	vpc, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "VpcManager.FetchById(%s)", self.VpcId)
	}
	return vpc.(*SVpc), nil
}

func (self *SVpcPeeringConnection) GetPeerVpc() (*SVpc, error) {
	vpc, err := VpcManager.FetchById(self.PeerVpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "VpcManager.FetchById(%s)", self.VpcId)
	}
	return vpc.(*SVpc), nil
}
