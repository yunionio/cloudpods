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
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SAdminSharableVirtualResourceBase struct {
	SSharableVirtualResourceBase
	Records string `charset:"ascii" list:"user" create:"optional" update:"user"`
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

func (manager *SAdminSharableVirtualResourceBaseManager) ValidateCreateData(man IAdminSharableVirtualModelManager, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
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
