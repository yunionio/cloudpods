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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	NetworkschedtagManager *SNetworkschedtagManager
	_                      ISchedtagJointModel = new(SNetworkschedtag)
)

func init() {
	db.InitManager(func() {
		NetworkschedtagManager = &SNetworkschedtagManager{
			SSchedtagJointsManager: NewSchedtagJointsManager(
				SNetworkschedtag{},
				"schedtag_networks_tbl",
				"schedtagnetwork",
				"schedtagnetworks",
				NetworkManager,
			),
		}
		NetworkschedtagManager.SetVirtualObject(NetworkschedtagManager)
	})
}

// +onecloud:swagger-gen-ignore
type SNetworkschedtagManager struct {
	*SSchedtagJointsManager
	resourceBaseManager SNetworkResourceBaseManager
}

// +onecloud:model-api-gen
type SNetworkschedtag struct {
	SSchedtagJointsBase

	NetworkId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SNetworkschedtagManager) GetMasterFieldName() string {
	return "network_id"
}

func (s *SNetworkschedtag) GetResourceId() string {
	return s.NetworkId
}

func (s *SNetworkschedtag) GetDetails(base api.SchedtagJointResourceDetails, resourceName string, isList bool) interface{} {
	out := api.NetworkschedtagDetails{
		SchedtagJointResourceDetails: base,
	}
	out.Network = resourceName
	return out
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
	q, err = manager.resourceBaseManager.ListItemFilter(ctx, q, userCred, query.NetworkFilterListInput)
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
	q, err = manager.resourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.NetworkFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNetworkResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}
