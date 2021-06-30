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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGroupnetworkManager struct {
	SGroupJointsManager
	SNetworkResourceBaseManager
}

var GroupnetworkManager *SGroupnetworkManager

func init() {
	db.InitManager(func() {
		GroupnetworkManager = &SGroupnetworkManager{
			SGroupJointsManager: NewGroupJointsManager(
				SGroupnetwork{},
				"groupnetworks_tbl",
				"groupnetwork",
				"groupnetworks",
				NetworkManager,
			),
		}
		GroupnetworkManager.SetVirtualObject(GroupnetworkManager)
	})
}

type SGroupnetwork struct {
	SGroupJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	IpAddr string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)
	// # ip6_addr = Column(VARCHAR(64, charset='ascii'), nullable=True)

	Index int8 `nullable:"false" default:"0" list:"user" list:"user" update:"user" create:"optional"` // Column(TINYINT, nullable=False, default=0)

	EipId string `width:"36" charset:"ascii" nullable:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=True)
}

func (manager *SGroupnetworkManager) GetSlaveFieldName() string {
	return "network_id"
}

func (manager *SGroupnetworkManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GroupnetworkDetails {
	rows := make([]api.GroupnetworkDetails, len(objs))

	groupRows := manager.SGroupJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	netIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.GroupnetworkDetails{
			GroupJointResourceDetails: groupRows[i],
		}
		netIds[i] = objs[i].(*SGroupnetwork).NetworkId
	}

	netIdMaps, err := db.FetchIdNameMap2(NetworkManager, netIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := netIdMaps[netIds[i]]; ok {
			rows[i].Network = name
		}
	}

	return rows
}

func (self *SGroupnetwork) GetNetwork() *SNetwork {
	obj, err := NetworkManager.FetchById(self.NetworkId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return obj.(*SNetwork)
}

func (self *SGroupnetwork) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGroupnetwork) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SGroupnetworkManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GroupnetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGroupJointsManager.ListItemFilter(ctx, q, userCred, query.GroupJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGroupJointsManager.ListItemFilter")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}

	if len(query.IpAddr) > 0 {
		q = q.In("ip_addr", query.IpAddr)
	}

	return q, nil
}

func (manager *SGroupnetworkManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GroupnetworkListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGroupJointsManager.OrderByExtraFields(ctx, q, userCred, query.GroupJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGroupJointsManager.OrderByExtraFields")
	}
	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SGroupnetworkManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGroupJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SGroupJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SNetworkResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SNetworkResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
