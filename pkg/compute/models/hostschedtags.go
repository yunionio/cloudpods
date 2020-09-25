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

type SHostschedtagManager struct {
	*SSchedtagJointsManager
	SHostResourceBaseManager
}

var HostschedtagManager *SHostschedtagManager

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

type SHostschedtag struct {
	SSchedtagJointsBase

	HostId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (manager *SHostschedtagManager) GetMasterFieldName() string {
	return "host_id"
}

func (self *SHostschedtag) GetHost() *SHost {
	return self.Master().(*SHost)
}

func (self *SHostschedtag) GetHosts() ([]SHost, error) {
	hosts := []SHost{}
	err := self.GetSchedtag().GetObjects(&hosts)
	return hosts, err
}

func (self *SHostschedtag) Master() db.IStandaloneModel {
	return self.SSchedtagJointsBase.master(self)
}

func (self *SHostschedtag) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.HostschedtagDetails, error) {
	return api.HostschedtagDetails{}, nil
}

func (manager *SHostschedtagManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HostschedtagDetails {
	rows := make([]api.HostschedtagDetails, len(objs))

	schedRows := manager.SSchedtagJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	hostIds := make([]string, len(rows))
	for i := range rows {
		rows[i] = api.HostschedtagDetails{
			SchedtagJointResourceDetails: schedRows[i],
		}
		hostIds[i] = objs[i].(*SHostschedtag).HostId
	}

	hostIdMaps, err := db.FetchIdNameMap2(HostManager, hostIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 hostIds fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := hostIdMaps[hostIds[i]]; ok {
			rows[i].Host = name
		}
	}

	return rows
}

func (self *SHostschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SSchedtagJointsBase.delete(self, ctx, userCred)
}

func (self *SHostschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SSchedtagJointsBase.detach(self, ctx, userCred)
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
	q, err = manager.SHostResourceBaseManager.ListItemFilter(ctx, q, userCred, query.HostFilterListInput)
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
	q, err = manager.SHostResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.HostFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SHostResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}
