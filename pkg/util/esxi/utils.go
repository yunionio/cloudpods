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

package esxi

import (
	"context"
	"reflect"

	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/log"
)

func Properties(c *vim25.Client, ctx context.Context, r types.ManagedObjectReference, ps []string, dst interface{}) error {
	return property.DefaultCollector(c).RetrieveOne(ctx, r, ps, dst)
}

func moRefId(ref types.ManagedObjectReference) string {
	return ref.Value
}

func moRefType(ref types.ManagedObjectReference) string {
	return ref.Type
}

func FetchAnonymousFieldValue(val interface{}, target interface{}) bool {
	return fetchAnonymousFieldValue(reflect.Indirect(reflect.ValueOf(val)),
		reflect.Indirect(reflect.ValueOf(target)))
}

func fetchAnonymousFieldValue(value reflect.Value, target reflect.Value) bool {
	for i := 0; i < value.NumField(); i += 1 {
		fieldValue := value.Field(i)
		fieldStruct := value.Type().Field(i)
		if fieldStruct.Anonymous && fieldStruct.Type.Kind() == reflect.Struct {
			if fieldStruct.Type == target.Type() {
				target.Set(fieldValue)
				return true
			}
			succ := fetchAnonymousFieldValue(fieldValue, target)
			if succ {
				return true
			}
		}
	}
	return false
}

func reverseArray(array interface{}) {
	arrayValue := reflect.Indirect(reflect.ValueOf(array))
	if arrayValue.Kind() != reflect.Slice && arrayValue.Kind() != reflect.Array {
		log.Errorf("reverse non array or slice")
		return
	}
	tmp := reflect.Indirect(reflect.New(arrayValue.Type().Elem()))
	for i, j := 0, arrayValue.Len()-1; i < j; i, j = i+1, j-1 {
		tmpi := arrayValue.Index(i)
		tmpj := arrayValue.Index(j)
		tmp.Set(tmpi)
		tmpi.Set(tmpj)
		tmpj.Set(tmp)
	}
}
