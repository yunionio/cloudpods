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
	"reflect"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type ISpecModelManager interface {
	db.IStandaloneModelManager
	GetSpecShouldCheckStatus(query *jsonutils.JSONDict) (bool, error)
	GetSpecIdent(spec *jsonutils.JSONDict) []string
}

type ISpecModel interface {
	db.IStandaloneModel
	GetSpec(statusCheck bool) *jsonutils.JSONDict
}

func GetAllModelSpecs(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	mans := []ISpecModelManager{HostManager, IsolatedDeviceManager, GuestManager}
	return GetModelsSpecs(ctx, userCred, query, mans...)
}

func GetModelsSpecs(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict, managers ...ISpecModelManager) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	for _, man := range managers {
		spec, err := getModelSpecs(man, ctx, userCred, query)
		if err != nil {
			return nil, err
		}
		ret.Add(spec, man.KeywordPlural())
	}
	return ret, nil
}

func GetHostSpecs(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	return getModelSpecs(HostManager, ctx, userCred, query)
}

func GetIsolatedDeviceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	return getModelSpecs(IsolatedDeviceManager, ctx, userCred, query)
}

func GetServerSpecs(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	return getModelSpecs(GuestManager, ctx, userCred, query)
}

func GetSpecIdentKey(keys []string) string {
	sort.Strings(keys)
	return strings.Join(keys, "/")
}

func GetModelSpec(manager ISpecModelManager, model ISpecModel) (jsonutils.JSONObject, error) {
	spec := model.GetSpec(false)
	specKey := GetSpecIdentKey(manager.GetSpecIdent(spec))
	spec.Add(jsonutils.NewString(specKey), "spec_key")
	return spec, nil
}

func getModelSpecs(manager ISpecModelManager, ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	items, err := ListItems(manager, ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	retDict := jsonutils.NewDict()
	statusCheck, err := manager.GetSpecShouldCheckStatus(query)
	if err != nil {
		return nil, err
	}
	for _, obj := range items {
		specObj := obj.(ISpecModel)
		spec := specObj.GetSpec(statusCheck)
		if spec == nil {
			continue
		}
		specKeys := manager.GetSpecIdent(spec)
		specKey := GetSpecIdentKey(specKeys)
		if oldSpec, _ := retDict.Get(specKey); oldSpec == nil {
			spec.Add(jsonutils.NewInt(1), "count")
			retDict.Add(spec, specKey)
		} else {
			count, _ := oldSpec.Int("count")
			oldSpec.(*jsonutils.JSONDict).Set("count", jsonutils.NewInt(count+1))
			retDict.Set(specKey, oldSpec)
		}
	}
	return retDict, nil
}

func ListItems(manager db.IModelManager, ctx context.Context, userCred mcclient.TokenCredential, queryDict *jsonutils.JSONDict) ([]ISpecModel, error) {
	q := manager.Query()
	queryDict, err := manager.ValidateListConditions(ctx, userCred, queryDict)
	if err != nil {
		return nil, err
	}
	q, err = db.ListItemQueryFilters(manager, ctx, q, userCred, queryDict, policy.PolicyActionList)
	if err != nil {
		return nil, err
	}
	customizeFilters, err := manager.CustomizeFilterList(ctx, q, userCred, queryDict)
	if err != nil {
		return nil, err
	}
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make(map[string]ISpecModel, 0)
	tmpLists := make([]jsonutils.JSONObject, 0)
	for rows.Next() {
		item, err := db.NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		itemInitValue := reflect.Indirect(reflect.ValueOf(item))
		item, err = db.NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		itemValue := reflect.Indirect(reflect.ValueOf(item))
		itemValue.Set(itemInitValue)
		err = q.Row2Struct(rows, item)
		if err != nil {
			return nil, err
		}
		records[item.GetId()] = item.(ISpecModel)
		tmpLists = append(tmpLists, jsonutils.Marshal(item).(*jsonutils.JSONDict))
	}
	if !customizeFilters.IsEmpty() {
		tmpLists, err = customizeFilters.DoApply(tmpLists)
		if err != nil {
			return nil, err
		}
	}
	items := make([]ISpecModel, 0)
	for _, obj := range tmpLists {
		id, _ := obj.GetString("id")
		items = append(items, records[id])
	}

	return items, nil
}
