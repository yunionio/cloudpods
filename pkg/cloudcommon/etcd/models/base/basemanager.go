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

package base

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/object"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
)

var (
	ErrNotJson = errors.New("Not a JSON")
)

const (
	MODEL_KEY_SEPARATOR    = "/"
	GLOBAL_MODEL_NAMESPACE = "models"
)

type SEtcdBaseModelManager struct {
	object.SObject

	keyword       string
	keywordPlural string
	dataType      reflect.Type
}

func NewEtcdBaseModelManager(model IEtcdModel, keyword string, keywordPlural string) SEtcdBaseModelManager {
	return SEtcdBaseModelManager{
		keyword:       keyword,
		keywordPlural: keywordPlural,
		dataType:      reflect.Indirect(reflect.ValueOf(model)).Type(),
	}
}

func (manager *SEtcdBaseModelManager) Keyword() string {
	return manager.keyword
}

func (manager *SEtcdBaseModelManager) KeywordPlural() string {
	return manager.keywordPlural
}

func (manager *SEtcdBaseModelManager) CustomizeHandlerInfo(handler *appsrv.SHandlerInfo) {
	// do nothing
}

func (manager *SEtcdBaseModelManager) FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (manager *SEtcdBaseModelManager) FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return nil, nil
}

func path2key(segs []string) string {
	return strings.Join(segs, MODEL_KEY_SEPARATOR)
}

func key2Path(key string) []string {
	return strings.Split(key, MODEL_KEY_SEPARATOR)
}

func (manager *SEtcdBaseModelManager) managerKey() string {
	return path2key([]string{GLOBAL_MODEL_NAMESPACE, manager.keywordPlural})
}

func (manager *SEtcdBaseModelManager) modelKey(idstr string) string {
	return path2key([]string{GLOBAL_MODEL_NAMESPACE, manager.keywordPlural, idstr})
}

func (manager *SEtcdBaseModelManager) key2Id(key string) string {
	segs := key2Path(key)
	if len(segs) >= 3 && segs[0] == GLOBAL_MODEL_NAMESPACE && segs[1] == manager.keywordPlural {
		return segs[2]
	} else {
		return ""
	}
}

func (manager *SEtcdBaseModelManager) Allocate() IEtcdModel {
	return reflect.New(manager.dataType).Interface().(IEtcdModel)
}

func (manager *SEtcdBaseModelManager) AllJson(ctx context.Context) ([]jsonutils.JSONObject, error) {
	prefix := manager.managerKey()
	kvs, err := etcd.Default().List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	dest := make([]jsonutils.JSONObject, 0)

	for i := range kvs {
		key := string(kvs[i].Key)
		jsonVal, err := jsonutils.Parse(kvs[i].Value)
		if err == nil {
			jsonDict, ok := jsonVal.(*jsonutils.JSONDict)
			if ok {
				idStr := manager.key2Id(key)
				if len(idStr) > 0 {
					jsonDict.Set("id", jsonutils.NewString(idStr))
					dest = append(dest, jsonDict)
				} else {
					log.Warningf("invalid key %s", key)
				}
			} else {
				log.Warningf("value %s is not a json dict??", jsonVal)
			}
		} else {
			log.Warningf("fail to json decode %s: %s", string(kvs[i].Value), err)
		}
	}

	return dest, nil
}

func (manager *SEtcdBaseModelManager) GetJson(ctx context.Context, idstr string) (jsonutils.JSONObject, error) {
	prefix := manager.modelKey(idstr)
	val, err := etcd.Default().Get(ctx, prefix)
	if err != nil {
		return nil, err
	}
	jsonVal, err := jsonutils.Parse(val)
	if err != nil {
		return nil, err
	}
	jsonDict, ok := jsonVal.(*jsonutils.JSONDict)
	if !ok {
		return nil, ErrNotJson
	}
	jsonDict.Set("id", jsonutils.NewString(idstr))
	return jsonDict, nil
}

func (manager *SEtcdBaseModelManager) json2Model(jsonObj jsonutils.JSONObject, model IEtcdModel) error {
	err := jsonObj.Unmarshal(model)
	if err != nil {
		return err
	}
	model.SetModelManager(manager, model)
	return nil
}

func (manager *SEtcdBaseModelManager) Get(ctx context.Context, idstr string, model IEtcdModel) error {
	jsonVal, err := manager.GetJson(ctx, idstr)
	if err != nil {
		return err
	}
	return manager.json2Model(jsonVal, model)
}

func (manager *SEtcdBaseModelManager) jsonArray2Models(jsonArray []jsonutils.JSONObject, dest interface{}) error {
	arrayType := reflect.TypeOf(dest).Elem()

	if arrayType.Kind() != reflect.Array && arrayType.Kind() != reflect.Slice {
		return fmt.Errorf("dest is not an array or slice")
	}
	elemType := arrayType.Elem()

	arrayValue := reflect.ValueOf(dest).Elem()
	for i := 0; i < len(jsonArray); i += 1 {
		elemPtrValue := reflect.New(elemType)
		model := elemPtrValue.Interface().(IEtcdModel)
		err := manager.json2Model(jsonArray[i], model)
		if err != nil {
			log.Errorf("fail to convert 2 model %s", err)
			return err
		}
		elemValue := reflect.Indirect(elemPtrValue)
		newArray := reflect.Append(arrayValue, elemValue)
		arrayValue.Set(newArray)
	}

	return nil
}

func (manager *SEtcdBaseModelManager) All(ctx context.Context, dest interface{}) error {
	jsonArray, err := manager.AllJson(ctx)
	if err != nil {
		return err
	}
	return manager.jsonArray2Models(jsonArray, dest)
}

func (manager *SEtcdBaseModelManager) Save(ctx context.Context, model IEtcdModel) error {
	if len(model.GetId()) == 0 {
		model.SetId(stringutils.UUID4())
	}
	prefix := manager.modelKey(model.GetId())
	return etcd.Default().Put(ctx, prefix, jsonutils.Marshal(model).String())
}

func (manager *SEtcdBaseModelManager) Session(ctx context.Context, model IEtcdModel) error {
	if len(model.GetId()) == 0 {
		model.SetId(stringutils.UUID4())
	}
	prefix := manager.modelKey(model.GetId())
	return etcd.Default().PutSession(ctx, prefix, jsonutils.Marshal(model).String())
}

func (manager *SEtcdBaseModelManager) Delete(ctx context.Context, model IEtcdModel) error {
	prefix := manager.modelKey(model.GetId())
	_, err := etcd.Default().Delete(ctx, prefix)
	if err != nil && err != etcd.ErrNoSuchKey {
		return err
	}
	return nil
}

func (manager *SEtcdBaseModelManager) Watch(ctx context.Context,
	onCreate etcd.TEtcdCreateEventFunc,
	onModify etcd.TEtcdModifyEventFunc,
	onDelete etcd.TEtcdDeleteEventFunc,
) {
	prefix := manager.managerKey()
	etcd.Default().Watch(ctx, prefix, onCreate, onModify, onDelete)
}
