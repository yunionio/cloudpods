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
	HostschedtagManager *SHostschedtagManager
	_                   ISchedtagJointModel = new(SHostschedtag)
)

func init() {
	db.InitManager(func() {
		HostschedtagManager = &SHostschedtagManager{
			SSchedtagJointsManager: NewSchedtagJointsManager(
				SHostschedtag{},
				"aggregate_hosts_tbl",
				"schedtaghost",
				"schedtaghosts",
				HostManager,
			),
		}
		HostschedtagManager.SetVirtualObject(HostschedtagManager)
	})
}

// +onecloud:swagger-gen-ignore
type SHostschedtagManager struct {
	*SSchedtagJointsManager
	resourceBaseManager SHostResourceBaseManager
}

// +onecloud:model-api-gen
type SHostschedtag struct {
	SSchedtagJointsBase

	HostId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SHostschedtagManager) GetMasterFieldName() string {
	return "host_id"
}

func (self *SHostschedtag) GetDetails(base api.SchedtagJointResourceDetails, resourceName string, isList bool) interface{} {
	out := api.HostschedtagDetails{
		SchedtagJointResourceDetails: base,
	}
	out.Host = resourceName
	return out
}

func (self *SHostschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SSchedtagJointsBase.delete(self, ctx, userCred)
}

func (self *SHostschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SSchedtagJointsBase.detach(self, ctx, userCred)
}

func (self *SHostschedtag) GetResourceId() string {
	return self.HostId
}

func (manager *SHostschedtagManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostschedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSchedtagJointsManager.ListItemFilter(ctx, q, userCred, query.SchedtagJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.ListItemFilter")
	}
	q, err = manager.resourceBaseManager.ListItemFilter(ctx, q, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SHostschedtagManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.HostschedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSchedtagJointsManager.OrderByExtraFields(ctx, q, userCred, query.SchedtagJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.OrderByExtraFields")
	}
	q, err = manager.resourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}
