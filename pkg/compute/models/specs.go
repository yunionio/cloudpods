package models

import (
	"context"
	"reflect"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type ISpecModelManager interface {
	db.IStandaloneModelManager
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

func getModelSpecs(manager ISpecModelManager, ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	items, err := ListItems(manager, ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	retDict := jsonutils.NewDict()
	for _, obj := range items {
		specObj := obj.(ISpecModel)
		spec := specObj.GetSpec(true)
		if spec == nil {
			continue
		}
		specKeys := manager.GetSpecIdent(spec)
		sort.Strings(specKeys)
		specKey := strings.Join(specKeys, "/")
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
	q, err = db.ListItemQueryFilters(manager, ctx, q, userCred, queryDict)
	if err != nil {
		return nil, err
	}
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	items := make([]ISpecModel, 0)
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
		items = append(items, item.(ISpecModel))
	}
	return items, err
}
