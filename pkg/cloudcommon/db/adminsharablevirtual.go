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

package db

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAdminSharableVirtualResourceBase struct {
	SSharableVirtualResourceBase
	Records string `charset:"utf8" list:"user" create:"optional" update:"user"`
}

type SAdminSharableVirtualResourceBaseManager struct {
	SSharableVirtualResourceBaseManager
	RecordsSeparator string
	RecordsLimit     int
}

func NewAdminSharableVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SAdminSharableVirtualResourceBaseManager {
	manager := SAdminSharableVirtualResourceBaseManager{SSharableVirtualResourceBaseManager: NewSharableVirtualResourceBaseManager(dt, tableName, keyword, keywordPlural)}
	manager.RecordsSeparator = ";"
	manager.RecordsLimit = 0
	return manager
}

func (manager *SAdminSharableVirtualResourceBaseManager) GetIAdminSharableVirtualModelManager() IAdminSharableVirtualModelManager {
	return manager.GetVirtualObject().(IAdminSharableVirtualModelManager)
}

func (manager *SAdminSharableVirtualResourceBaseManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input apis.AdminSharableVirtualResourceBaseCreateInput,
) (apis.AdminSharableVirtualResourceBaseCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
	}
	return input, nil
}

func (manager *SAdminSharableVirtualResourceBaseManager) ValidateRecordsData(man IAdminSharableVirtualModelManager, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	records, err := man.ParseInputInfo(data)
	if err != nil {
		return nil, err
	}
	data.Add(jsonutils.NewString(strings.Join(records, man.GetRecordsSeparator())), "records")
	return data, nil
}

func (model *SAdminSharableVirtualResourceBase) SetInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	man IAdminSharableVirtualModelManager,
	obj IAdminSharableVirtualModel,
	data *jsonutils.JSONDict,
) error {
	records, err := man.ParseInputInfo(data)
	if err != nil {
		return err
	}
	oldRecs := obj.GetInfo()
	isChanged := false
	if len(records) != len(oldRecs) {
		isChanged = true
	} else {
		for _, rec := range records {
			if !utils.IsInStringArray(rec, oldRecs) {
				isChanged = true
				break
			}
		}
	}
	if isChanged {
		return model.setInfo(ctx, userCred, man, records)
	}
	return nil
}

func (model *SAdminSharableVirtualResourceBase) AddInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	man IAdminSharableVirtualModelManager,
	obj IAdminSharableVirtualModel,
	data jsonutils.JSONObject,
) error {
	records, err := man.ParseInputInfo(data.(*jsonutils.JSONDict))
	if err != nil {
		return err
	}
	oldRecs := obj.GetInfo()
	adds := []string{}
	for _, r := range records {
		if !utils.IsInStringArray(r, oldRecs) {
			oldRecs = append(oldRecs, r)
			adds = append(adds, r)
		}
	}
	if len(adds) > 0 {
		return model.setInfo(ctx, userCred, man, oldRecs)
	}
	return nil
}

func (model *SAdminSharableVirtualResourceBase) RemoveInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	man IAdminSharableVirtualModelManager,
	obj IAdminSharableVirtualModel,
	data jsonutils.JSONObject,
	allowEmpty bool,
) error {
	records, err := man.ParseInputInfo(data.(*jsonutils.JSONDict))
	if err != nil {
		return err
	}
	oldRecs := obj.GetInfo()
	removes := []string{}
	for _, rec := range records {
		if ok, idx := utils.InStringArray(rec, oldRecs); ok {
			oldRecs = append(oldRecs[:idx], oldRecs[idx+1:]...) // remove record
			removes = append(removes, rec)
		}
	}
	if len(oldRecs) == 0 && !allowEmpty {
		return httperrors.NewNotAcceptableError("Not allow empty records")
	}
	if len(removes) > 0 {
		return model.setInfo(ctx, userCred, man, oldRecs)
	}
	return nil
}

func (model *SAdminSharableVirtualResourceBase) setInfo(ctx context.Context,
	userCred mcclient.TokenCredential,
	man IAdminSharableVirtualModelManager,
	records []string,
) error {
	if man.GetRecordsLimit() > 0 && len(records) > man.GetRecordsLimit() {
		return httperrors.NewNotAcceptableError("Records limit exceeded.")
	}
	diff, err := Update(model, func() error {
		model.Records = strings.Join(records, man.GetRecordsSeparator())
		return nil
	})
	if err != nil {
		return err
	}
	OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
	return err
}

func (model *SAdminSharableVirtualResourceBase) GetIAdminSharableVirtualModel() IAdminSharableVirtualModel {
	return model.GetVirtualObject().(IAdminSharableVirtualModel)
}

func (manager *SAdminSharableVirtualResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.AdminSharableVirtualResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SAdminSharableVirtualResourceBaseManager) QueryDistinctExtraFields(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (manager *SAdminSharableVirtualResourceBaseManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.AdminSharableVirtualResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SAdminSharableVirtualResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []apis.AdminSharableVirtualResourceDetails {
	rows := make([]apis.AdminSharableVirtualResourceDetails, len(objs))
	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = apis.AdminSharableVirtualResourceDetails{
			SharableVirtualResourceDetails: virtRows[i],
		}
	}
	return rows
}

func (model *SAdminSharableVirtualResourceBase) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input apis.AdminSharableVirtualResourceBaseUpdateInput,
) (apis.AdminSharableVirtualResourceBaseUpdateInput, error) {
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = model.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}
