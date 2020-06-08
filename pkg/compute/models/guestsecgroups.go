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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

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

type SGuestsecgroup struct {
	SGuestJointsBase

	SSecurityGroupResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	// SecgroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
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

func (manager *SGuestsecgroupManager) newGuestSecgroup(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, secgroup *SSecurityGroup) (*SGuestsecgroup, error) {
	q := manager.Query()
	q = q.Equals("guest_id", guest.Id).Equals("secgroup_id", secgroup.Id)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("security group %s has already been assigned to guest %s", secgroup.Name, guest.Name)
	}

	gs := SGuestsecgroup{}
	gs.SecgroupId = secgroup.Id
	gs.SetModelManager(manager, &gs)
	gs.GuestId = guest.Id

	lockman.LockObject(ctx, secgroup)
	defer lockman.ReleaseObject(ctx, secgroup)

	return &gs, manager.TableSpec().Insert(ctx, &gs)
}

func (manager *SGuestsecgroupManager) GetGuestSecgroups(guest *SGuest, secgroup *SSecurityGroup) ([]SGuestsecgroup, error) {
	guestsecgroups := []SGuestsecgroup{}
	q := manager.Query()
	if guest != nil {
		q = q.Equals("guest_id", guest.Id)
	}
	if secgroup != nil {
		q = q.Equals("secgroup_id", secgroup.Id)
	}
	if err := db.FetchModelObjects(manager, q, &guestsecgroups); err != nil {
		return nil, err
	}
	return guestsecgroups, nil
}

func (manager *SGuestsecgroupManager) DeleteGuestSecgroup(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, secgroup *SSecurityGroup) error {
	gss := []SGuestsecgroup{}
	q := manager.Query()
	if guest != nil {
		q = q.Equals("guest_id", guest.Id)
	}
	if secgroup != nil {
		q = q.Equals("secgroup_id", secgroup.Id)
	}
	if err := db.FetchModelObjects(manager, q, &gss); err != nil {
		return err
	}
	for _, gs := range gss {
		if err := gs.Delete(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuestsecgroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
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

func (self *SGuestsecgroup) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.GuestsecgroupDetails, error) {
	return api.GuestsecgroupDetails{}, nil
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
