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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGlobalVpcResourceBase struct {
	GlobalvpcId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"globalvpc_id"`
}

type SGlobalVpcResourceBaseManager struct{}

func ValidateGlobalvpcResourceInput(ctx context.Context, userCred mcclient.TokenCredential, input api.GlobalVpcResourceInput) (*SGlobalVpc, api.GlobalVpcResourceInput, error) {
	gvpcObj, err := GlobalVpcManager.FetchByIdOrName(ctx, userCred, input.GlobalvpcId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, input, errors.Wrapf(httperrors.ErrResourceNotFound, "%s %s", GlobalVpcManager.Keyword(), input.GlobalvpcId)
		} else {
			return nil, input, errors.Wrap(err, "GlobalVpcManager.FetchByIdOrName")
		}
	}
	input.GlobalvpcId = gvpcObj.GetId()
	return gvpcObj.(*SGlobalVpc), input, nil
}

func (self *SGlobalVpcResourceBase) GetGlobalVpc() (*SGlobalVpc, error) {
	if len(self.GlobalvpcId) == 0 {
		return nil, nil
	}
	gv, err := GlobalVpcManager.FetchById(self.GlobalvpcId)
	if err != nil {
		return nil, err
	}
	return gv.(*SGlobalVpc), nil
}

func (manager *SGlobalVpcResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GlobalVpcResourceInfo {
	rows := make([]api.GlobalVpcResourceInfo, len(objs))
	globalVpcIds := make([]string, len(objs))
	for i := range objs {
		var base *SGlobalVpcResourceBase
		reflectutils.FindAnonymouStructPointer(objs[i], &base)
		if base != nil {
			globalVpcIds[i] = base.GlobalvpcId
		}
	}
	globalVpcs := make(map[string]SGlobalVpc)
	err := db.FetchStandaloneObjectsByIds(GlobalVpcManager, globalVpcIds, globalVpcs)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail %s", err)
		return nil
	}
	for i := range rows {
		rows[i] = api.GlobalVpcResourceInfo{}
		if _, ok := globalVpcs[globalVpcIds[i]]; ok {
			rows[i].Globalvpc = globalVpcs[globalVpcIds[i]].Name
		}
	}
	return rows
}

func (manager *SGlobalVpcResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GlobalVpcResourceListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.GlobalvpcId) > 0 {
		globalVpcObj, _, err := ValidateGlobalvpcResourceInput(ctx, userCred, query.GlobalVpcResourceInput)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateGlobalvpcResourceInput")
		}
		q = q.Equals("globalvpc_id", globalVpcObj.GetId())
	}
	return q, nil
}

func (manager *SGlobalVpcResourceBaseManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	if field == "globalvpc" {
		globalvpcs := GlobalVpcManager.Query("name", "id").Distinct().SubQuery()
		q.AppendField(globalvpcs.Field("name", field))
		q = q.Join(globalvpcs, sqlchemy.Equals(q.Field("globalvpc_id"), globalvpcs.Field("id")))
		q.GroupBy(globalvpcs.Field("name"))
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SGlobalVpcResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GlobalVpcResourceListInput,
) (*sqlchemy.SQuery, error) {
	q = db.OrderByStandaloneResourceName(q, GlobalVpcManager, "globalvpc_id", query.OrderByGlobalvpc)
	return q, nil
}

func (manager *SGlobalVpcResourceBaseManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	if keys.ContainsAny(manager.GetExportKeys()...) {
		subq := GlobalVpcManager.Query("id", "name").SubQuery()
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field("globalvpc_id"), subq.Field("id")))
		if keys.Contains("globalvpc") {
			q = q.AppendField(subq.Field("name", "globalvpc"))
		}
	}
	return q, nil
}

func (manager *SGlobalVpcResourceBaseManager) GetExportKeys() []string {
	return []string{"globalvpc"}
}
