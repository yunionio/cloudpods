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

// +onecloud:swagger-gen-ignore
type SGuestsecgroupManager struct {
	SGuestJointsManager
	SSecurityGroupResourceBaseManager
}

var GuestsecgroupManager *SGuestsecgroupManager

func init() {
	db.InitManager(func() {
		GuestsecgroupManager = &SGuestsecgroupManager{
			SGuestJointsManager: NewGuestJointsManager(
				SGuestsecgroup{},
				"guestsecgroups_tbl",
				"guestsecgroup",
				"guestsecgroups",
				SecurityGroupManager,
			),
		}
		GuestsecgroupManager.SetVirtualObject(GuestsecgroupManager)
	})
}

// +onecloud:model-api-gen
type SGuestsecgroup struct {
	SGuestJointsBase

	SSecurityGroupResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
}

func (manager *SGuestsecgroupManager) GetSlaveFieldName() string {
	return "secgroup_id"
}

func (self *SGuestsecgroup) getSecgroup() *SSecurityGroup {
	secgrp, err := SecurityGroupManager.FetchById(self.SecgroupId)
	if err != nil {
		log.Errorf("failed to find secgroup %s", self.SecgroupId)
		return nil
	}
	secgroup := secgrp.(*SSecurityGroup)
	secgroup.SetModelManager(SecurityGroupManager, secgroup)
	return secgroup
}

func (self *SGuestsecgroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGuestsecgroup) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SGuestsecgroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestsecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.ListItemFilter(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemFilter")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SGuestsecgroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestsecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.OrderByExtraFields(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.OrderByExtraFields")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SGuestsecgroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SSecurityGroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SSecurityGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SGuestsecgroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GuestsecgroupDetails {
	rows := make([]api.GuestsecgroupDetails, len(objs))

	guestRows := manager.SGuestJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	secgroupIds := make([]string, len(rows))
	for i := range rows {
		rows[i].GuestJointResourceDetails = guestRows[i]
		secgroupIds[i] = objs[i].(*SGuestsecgroup).SecgroupId
	}

	secgroupIdMaps, err := db.FetchIdNameMap2(SecurityGroupManager, secgroupIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := secgroupIdMaps[secgroupIds[i]]; ok {
			rows[i].Secgroup = name
		}
	}

	return rows
}
