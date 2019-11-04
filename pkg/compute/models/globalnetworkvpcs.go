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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGlobalnetworkVpcManager struct {
	db.SJointResourceBaseManager
}

var GlobalnetworkVpcManager *SGlobalnetworkVpcManager

func init() {
	db.InitManager(func() {
		GlobalnetworkVpcManager = &SGlobalnetworkVpcManager{
			SJointResourceBaseManager: db.NewJointResourceBaseManager(
				SGlobalnetworkVpc{},
				"globalnetworkvpcs_tbl",
				"globalnetworkvpc",
				"globalnetworkvpcs",
				GlobalNetworkManager,
				VpcManager,
			),
		}
		GlobalnetworkVpcManager.SetVirtualObject(GlobalnetworkVpcManager)
	})
}

type SGlobalnetworkVpc struct {
	db.SJointResourceBase

	GlobalnetworkId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	VpcId           string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (manager *SGlobalnetworkVpcManager) GetMasterFieldName() string {
	return "globalnetwork_id"
}

func (manager *SGlobalnetworkVpcManager) GetSlaveFieldName() string {
	return "vpc_id"
}

func (joint *SGlobalnetworkVpc) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SGlobalnetworkVpc) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (manager *SGlobalnetworkVpcManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SGlobalnetworkVpcManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (manager *SGlobalnetworkVpcManager) AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model db.IStandaloneModel, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SGlobalnetworkVpcManager) AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master db.IStandaloneModel, slave db.IStandaloneModel) bool {
	return db.IsAdminAllowCreate(userCred, manager)
}

func (self *SGlobalnetworkVpc) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SGlobalnetworkVpc) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SGlobalnetworkVpc) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (self *SGlobalnetworkVpc) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (self *SGlobalnetworkVpc) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return self.SJointResourceBase.GetCustomizeColumns(ctx, userCred, query)
}

func (self *SGlobalnetworkVpc) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	return self.SJointResourceBase.GetExtraDetails(ctx, userCred, query)
}

func (manager *SGlobalnetworkVpcManager) NewGlobalnetworkVpc(vpc *SVpc, globalnetwork *SGlobalNetwork) error {
	q := manager.Query().Equals("vpc_id", vpc.Id).Equals("globalnetwork_id", globalnetwork.Id)
	count, err := q.CountWithError()
	if err != nil {
		return errors.Wrap(err, "CountWithError")
	}
	if count > 1 {
		return sqlchemy.ErrDuplicateEntry
	}
	if count == 1 {
		return nil
	}
	gv := &SGlobalnetworkVpc{}
	gv.SetModelManager(manager, gv)
	gv.VpcId = vpc.Id
	gv.GlobalnetworkId = globalnetwork.Id
	return manager.TableSpec().Insert(gv)
}
