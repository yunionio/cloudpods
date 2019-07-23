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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SResourceBase struct {
	db.SResourceBase

	CreateBy string `width:"128" charset:"ascii" nullable:"true" create:"optional"`
	UpdateBy string `width:"128" charset:"ascii" nullable:"true" update:"user"`
	DeleteBy string `width:"128" charset:"ascii" nullable:"true"`

	Remark jsonutils.JSONObject `get:"user"`
}

type SResourceBaseManager struct {
	db.SResourceBaseManager
}

type IResourceBaseModel interface {
	db.IModel
	GetIResourceBaseModel() IResourceBaseModel
	SetDeleteBy(string)
}

func NewResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SResourceBaseManager {
	return SResourceBaseManager{db.NewResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (self *SResourceBase) GetIResourceBaseModel() IResourceBaseModel {
	return self.GetVirtualObject().(IResourceBaseModel)
}

func (self *SResourceBase) SetDeleteBy(uid string) {
	self.DeleteBy = uid
}

func (self *SResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := self.SResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, err
	}
	data.Set("create_by", jsonutils.NewString(userCred.GetUserId()))
	return data, nil
}

func (self *SResourceBase) PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SResourceBase.PreUpdate(ctx, userCred, query, data)
	self.UpdateBy = userCred.GetUserId()
}

func (self *SResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	item := self.GetIResourceBaseModel()
	_, err := db.Update(item, func() error {
		item.SetDeleteBy(userCred.GetUserId())
		return item.MarkDelete()
	})
	if err != nil {
		msg := fmt.Sprintf("save update error %s", err)
		log.Errorf(msg)
		return httperrors.NewGeneralError(err)
	}
	if userCred != nil {
		db.OpsLog.LogEvent(self, db.ACT_DELETE, self.GetShortDesc(ctx), userCred)
	}
	return nil
}
