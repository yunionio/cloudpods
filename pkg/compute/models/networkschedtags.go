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

type SNetworkschedtagManager struct {
	*SSchedtagJointsManager
	SNetworkResourceBaseManager
}

var NetworkschedtagManager *SNetworkschedtagManager

func init() {
	db.InitManager(func() {
		NetworkschedtagManager = &SNetworkschedtagManager{
			SSchedtagJointsManager: NewSchedtagJointsManager(
				SNetworkschedtag{},
				"schedtag_networks_tbl",
				"schedtagnetwork",
				"schedtagnetworks",
				NetworkManager,
				SchedtagManager,
			),
		}
		NetworkschedtagManager.SetVirtualObject(NetworkschedtagManager)
	})
}

type SNetworkschedtag struct {
	SSchedtagJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SNetworkschedtagManager) GetSlaveFieldName() string {
	return "network_id"
}

func (s *SNetworkschedtag) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.NetworkschedtagDetails, error) {
	return api.NetworkschedtagDetails{}, nil
}

func (manager *SNetworkschedtagManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NetworkschedtagDetails {
	rows := make([]api.NetworkschedtagDetails, len(objs))

	schedRows := manager.SSchedtagJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	netIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.NetworkschedtagDetails{
			SchedtagJointResourceDetails: schedRows[i],
		}
		netIds[i] = objs[i].(*SNetworkschedtag).NetworkId
	}

	netIdMaps, err := db.FetchIdNameMap2(NetworkManager, netIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 netIds fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := netIdMaps[netIds[i]]; ok {
			rows[i].Network = name
		}
	}

	return rows
}

func (s *SNetworkschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return s.SSchedtagJointsBase.delete(s, ctx, userCred)
}

func (s *SNetworkschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return s.SSchedtagJointsBase.detach(s, ctx, userCred)
}

func (manager *SNetworkschedtagManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetworkschedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSchedtagJointsManager.ListItemFilter(ctx, q, userCred, query.SchedtagJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.ListItemFilter")
	}
	q, err = manager.SNetworkResourceBaseManager.ListItemFilter(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SNetworkschedtagManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.NetworkschedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSchedtagJointsManager.OrderByExtraFields(ctx, q, userCred, query.SchedtagJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.OrderByExtraFields")
	}
	q, err = manager.SNetworkResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}
