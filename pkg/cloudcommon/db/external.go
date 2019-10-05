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
	"database/sql"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SExternalizedResourceBase struct {
	ExternalId string `width:"256" charset:"utf8" index:"true" list:"user" create:"admin_optional" update:"admin"`
}

func (model SExternalizedResourceBase) GetExternalId() string {
	return model.ExternalId
}

func (model *SExternalizedResourceBase) SetExternalId(idStr string) {
	model.ExternalId = idStr
}

type IExternalizedModelManager interface {
	IModelManager
	FetchByExternalId(idStr string) IExternalizedModel
}

type IExternalizedModel interface {
	IModel

	GetExternalId() string
	SetExternalId(idStr string)
}

func SetExternalId(model IExternalizedModel, userCred mcclient.TokenCredential, idStr string) error {
	if model.GetExternalId() != idStr {
		diff, err := Update(model, func() error {
			model.SetExternalId(idStr)
			return nil
		})
		if err == nil {
			OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
		}
		return err
	}
	return nil
}

func FetchByExternalId(manager IModelManager, idStr string) (IExternalizedModel, error) {
	q := manager.Query().Equals("external_id", idStr)
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		obj, err := NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		err = q.First(obj)
		if err != nil {
			return nil, err
		} else {
			return obj.(IExternalizedModel), nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}
