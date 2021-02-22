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
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

type SDistinctFieldManager struct {
	SModelBaseManager
}

var DistinctFieldManager *SDistinctFieldManager

func init() {
	DistinctFieldManager = &SDistinctFieldManager{
		SModelBaseManager: NewModelBaseManager(
			SDistinctField{},
			"distinct_fields_tbl",
			"distinct_field",
			"distinct_fields",
		),
	}
	DistinctFieldManager.SetVirtualObject(DistinctFieldManager)
}

const (
	DISTINCT_FIELD_SEP = "::"
)

type SDistinctField struct {
	SModelBase

	// 资源类型
	// example: network
	ObjType string `width:"40" charset:"ascii" index:"true" list:"user" get:"user"`

	// 资源组合ID
	// example: obj_type::key::value
	Id string `width:"128" charset:"utf8" primary:"true" list:"user" get:"user"`

	// Distinct Field
	// exmaple: 部门
	Key string `width:"64" charset:"utf8" primary:"true" list:"user" get:"user"`

	// Distinct Value
	// example: 技术部
	Value string `charset:"utf8" list:"user" get:"user"`
}

func (manager *SDistinctFieldManager) GetObjectDistinctFields(objType string) ([]SDistinctField, error) {
	q := manager.Query().Equals("obj_type", objType)
	fields := []SDistinctField{}
	err := FetchModelObjects(manager, q, &fields)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchModelObjects")
	}
	return fields, nil
}

func (manager *SDistinctFieldManager) InsertOrUpdate(ctx context.Context, modelManager IModelManager, key, value string) error {
	if len(key) == 0 || len(value) == 0 {
		return fmt.Errorf("empty key or value")
	}
	distinct := &SDistinctField{
		ObjType: modelManager.Keyword(),
		Key:     key,
		Value:   value,
		Id:      modelManager.Keyword() + DISTINCT_FIELD_SEP + key + DISTINCT_FIELD_SEP + value,
	}
	distinct.SetModelManager(manager, distinct)
	err := manager.TableSpec().InsertOrUpdate(ctx, distinct)
	if err != nil {
		if errors.Cause(err) == sqlchemy.ErrUnexpectRowCount {
			return nil
		}
		return err
	}
	return nil
}
