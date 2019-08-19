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
	"database/sql"

	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type UUIDGenerator func() string

var (
	DefaultUUIDGenerator = stringutils.UUID4
)

type SStandaloneResourceBase struct {
	SResourceBase

	ID string `width:"128" charset:"ascii" primary:"true" create:"optional"`
}

func (model *SStandaloneResourceBase) BeforeInsert() {
	if len(model.ID) == 0 {
		model.ID = DefaultUUIDGenerator()
	}
}

type SStandaloneResourceBaseManager struct {
	SResourceBaseManager
}

func NewStandaloneResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SStandaloneResourceBaseManager {
	return SStandaloneResourceBaseManager{NewResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (manager *SStandaloneResourceBaseManager) GetIStandaloneModelManager() db.IStandaloneModelManager {
	return manager.GetVirtualObject().(db.IStandaloneModelManager)
}

func (manager *SStandaloneResourceBaseManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (manager *SStandaloneResourceBaseManager) FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.NotEquals("id", idStr)
}

func (manager *SStandaloneResourceBaseManager) FetchById(idStr string) (db.IModel, error) {
	return FetchById(manager.GetIStandaloneModelManager(), idStr)
}

func FetchById(manager db.IModelManager, idStr string) (db.IModel, error) {
	q := manager.Query()
	q = manager.FilterById(q, idStr)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		obj, err := db.NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(obj)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func (model *SStandaloneResourceBase) StandaloneModelManager() db.IStandaloneModelManager {
	return model.GetModelManager().(db.IStandaloneModelManager)
}

func (model *SStandaloneResourceBase) GetId() string {
	return model.ID
}

func (model *SStandaloneResourceBase) GetIStandaloneModel() db.IStandaloneModel {
	return model.GetVirtualObject().(db.IStandaloneModel)
}
