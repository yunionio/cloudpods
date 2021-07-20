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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SI18nResourceBase struct {
}

type SI18nResourceBaseManager struct {
}

func (man *SI18nResourceBaseManager) getSModelI18nTable(model db.IModel, table cloudprovider.SModelI18nTable) *db.SModelI18nTable {
	entries := make([]db.IModelI18nEntry, 0)
	for k, _ := range table {
		entries = append(entries, db.NewSModelI18nEntry(k, model, table[k]))
	}

	return db.NewSModelI18nTable(entries)
}

func (man *SI18nResourceBaseManager) SyncI18ns(ctx context.Context, userCred mcclient.TokenCredential, model db.IModel, table cloudprovider.SModelI18nTable) error {
	itable := man.getSModelI18nTable(model, table)
	_, _, r := db.I18nManager.SyncI18ns(ctx, userCred, model, itable)
	if r.IsError() {
		log.Errorf("SyncI18ns for %s %s result: %s", model.Keyword(), model.GetId(), r.Result())
		return errors.Wrap(r.AllError(), "SyncI18ns")
	}

	return nil
}

func (self *SI18nResourceBase) RemoveI18ns(ctx context.Context, userCred mcclient.TokenCredential, model db.IModel) error {
	err := db.I18nManager.RemoveI18ns(ctx, userCred, model)
	if err != nil {
		return errors.Wrap(err, "RemoveI18ns")
	}

	return nil
}

func (self *SI18nResourceBase) GetModelI18N(ctx context.Context, model db.IModel) *jsonutils.JSONDict {
	entries, err := db.I18nManager.GetModelI18N(ctx, model)
	if err != nil {
		log.Errorf("GetModelI18N error %s", err)
		return nil
	}

	_i18n := jsonutils.NewDict()
	for i := range entries {
		_i18n.Set(entries[i].KeyName, jsonutils.NewString(entries[i].Lookup(ctx)))
	}

	return _i18n
}

func (self *SI18nResourceBase) GetModelKeyI18N(ctx context.Context, model db.IModel, keyName string) (string, bool) {
	entries, err := db.I18nManager.GetModelKeyI18N(ctx, model, keyName)
	if err != nil {
		log.Errorf("GetModelKeyI18N error %s", err)
		return "", false
	}

	if len(entries) > 0 {
		return entries[0].Lookup(ctx), true
	}

	return "", false
}
