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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IModelI18nBase interface {
	GetKeyValue() string
	Lookup(tag i18n.Tag) string
}

type IModelI18nEntry interface {
	IModelI18nBase
	GetGlobalId() string
	GetObjType() string
	GetObjId() string
	GetKeyName() string
}

type IModelI18nTable interface {
	GetI18nEntries() map[string]IModelI18nEntry
}

type SModelI18nEntry struct {
	model        IModel
	i18nEntry    IModelI18nBase
	modelKeyName string
}

func NewSModelI18nEntry(keyName string, model IModel, i18nEntry IModelI18nBase) *SModelI18nEntry {
	return &SModelI18nEntry{model: model, modelKeyName: keyName, i18nEntry: i18nEntry}
}

func (in *SModelI18nEntry) GetObjType() string {
	return in.model.Keyword()
}

func (in *SModelI18nEntry) GetObjId() string {
	return in.model.GetId()
}

func (in *SModelI18nEntry) GetKeyName() string {
	return in.modelKeyName
}

func (in *SModelI18nEntry) GetKeyValue() string {
	return in.i18nEntry.GetKeyValue()
}

func (in *SModelI18nEntry) Lookup(tag i18n.Tag) string {
	return in.i18nEntry.Lookup(tag)
}

func (in *SModelI18nEntry) GetGlobalId() string {
	return in.model.Keyword() + "::" + in.model.GetId() + "::" + in.modelKeyName
}

type SModelI18nTable struct {
	i18nEntires []IModelI18nEntry
}

func NewSModelI18nTable(i18nEntires []IModelI18nEntry) *SModelI18nTable {
	return &SModelI18nTable{i18nEntires: i18nEntires}
}

func (t *SModelI18nTable) GetI18nEntries() map[string]IModelI18nEntry {
	ret := make(map[string]IModelI18nEntry, 0)
	for i := range t.i18nEntires {
		entry := t.i18nEntires[i]
		ret[entry.GetKeyValue()] = entry
	}

	return ret
}

type SI18nManager struct {
	SModelBaseManager
	// SStandaloneAnonResourceBaseManager
}

type SI18n struct {
	SModelBase
	// SStandaloneAnonResourceBase

	// 资源类型
	// example: network
	ObjType string `width:"40" charset:"ascii" list:"user" get:"user" primary:"true"`

	// 资源ID
	// example: 87321a70-1ecb-422a-8b0c-c9aa632a46a7
	ObjId string `width:"88" charset:"ascii" list:"user" get:"user" primary:"true"`

	// 资源KEY
	// exmaple: name
	KeyName string `width:"64" charset:"utf8" list:"user" get:"user" primary:"true"`

	// 资源原始值
	// example: 技术部
	KeyValue string `charset:"utf8" list:"user" get:"user"`

	// KeyValue 对应中文翻译
	Cn string `charset:"utf8" list:"user" get:"user"`

	// KeyValue 对应英文翻译
	En string `charset:"utf8" list:"user" get:"user"`
}

var I18nManager *SI18nManager

func init() {
	I18nManager = &SI18nManager{
		SModelBaseManager: NewModelBaseManager(
			SI18n{},
			"i18n2_tbl",
			"i18n",
			"i18ns",
		),
	}
	I18nManager.SetVirtualObject(I18nManager)
}

func (manager *SI18nManager) GetModelI18N(ctx context.Context, model IModel) ([]SI18n, error) {
	ret := []SI18n{}
	q := manager.Query().Equals("obj_type", model.Keyword()).Equals("obj_id", model.GetId())
	err := FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}

	return ret, err
}

func (manager *SI18nManager) GetModelKeyI18N(ctx context.Context, model IModel, keyName string) ([]SI18n, error) {
	ret := []SI18n{}
	q := manager.Query().Equals("obj_type", model.Keyword()).Equals("obj_id", model.GetId()).Equals("key_name", keyName)
	err := FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}

	return ret, err
}

func (manager *SI18nManager) getExternalI18nItems(ctx context.Context, table IModelI18nTable) []IModelI18nEntry {
	ret := make([]IModelI18nEntry, 0)
	items := table.GetI18nEntries()
	for k, _ := range items {
		ret = append(ret, items[k])
	}

	return ret
}

func (manager *SI18nManager) SyncI18ns(ctx context.Context, userCred mcclient.TokenCredential, model IModel, table IModelI18nTable) error {
	//// No need to lock SI18nManager, the resources has been lock in the upper layer - QIU Jian, 20210405
	extItems := manager.getExternalI18nItems(ctx, table)

	for i := range extItems {
		_, err := manager.newFromI18n(ctx, userCred, extItems[i])
		if err != nil {
			return errors.Wrap(err, "newFromI18n")
		}
	}

	return nil
}

func (manager *SI18nManager) newFromI18n(ctx context.Context, userCred mcclient.TokenCredential, entry IModelI18nEntry) (*SI18n, error) {
	in := SI18n{}
	in.SetModelManager(manager, &in)

	in.ObjId = entry.GetObjId()
	in.ObjType = entry.GetObjType()
	in.KeyName = entry.GetKeyName()
	in.KeyValue = entry.GetKeyValue()
	in.Cn = entry.Lookup(i18n.I18N_TAG_CHINESE)
	in.En = entry.Lookup(i18n.I18N_TAG_ENGLISH)

	err := manager.TableSpec().InsertOrUpdate(ctx, &in)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromI18n.Insert")
	}

	return &in, nil
}

func (in SI18n) GetExternalId() string {
	return in.ObjType + "::" + in.ObjId + "::" + in.KeyName
}

func (in *SI18n) Lookup(ctx context.Context) string {
	entry := i18n.NewTableEntry().CN(in.Cn).EN(in.En)
	if v, ok := entry.Lookup(ctx); ok {
		return v
	}

	return in.KeyValue
}
