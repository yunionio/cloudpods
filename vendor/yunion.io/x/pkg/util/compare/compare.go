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

package compare

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type valueElement struct {
	key   string
	value reflect.Value
}

type valueSet []valueElement

func (v valueSet) Len() int {
	return len(v)
}

func (v valueSet) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v valueSet) Less(i, j int) bool {
	return strings.Compare(v[i].key, v[j].key) < 0
}

func valueSet2Array(dbSet interface{}, field string) ([]valueElement, error) {
	dbSetValue := reflect.Indirect(reflect.ValueOf(dbSet))
	if dbSetValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input set is not a slice")
	}
	ret := make([]valueElement, dbSetValue.Len())
	for i := 0; i < dbSetValue.Len(); i += 1 {
		val := dbSetValue.Index(i)
		// log.Debugf("valueSet2Array %d %s", i, val)

		funcValue := val.MethodByName(field)
		if !funcValue.IsValid() || funcValue.IsNil() {
			return nil, fmt.Errorf("no such method %s", field)
		}
		outVals := funcValue.Call([]reflect.Value{})
		if len(outVals) != 1 {
			return nil, fmt.Errorf("invalid return value, not 1 string")
		}
		keyVal, ok := outVals[0].Interface().(string)
		if !ok {
			return nil, fmt.Errorf("invalid output value for %s", field)
		}
		ret[i] = valueElement{value: dbSetValue.Index(i), key: keyVal}
	}
	return ret, nil
}

type SCompareSet struct {
	DBFunc  string
	DBSet   interface{}
	ExtFunc string
	ExtSet  interface{}
}

func CompareSetsFunc(cs SCompareSet, removed interface{}, commonDB interface{}, commonExt interface{}, added interface{}, duplicated interface{}) error {
	dbSetArray, err := valueSet2Array(cs.DBSet, cs.DBFunc)
	if err != nil {
		return err
	}
	extSetArray, err := valueSet2Array(cs.ExtSet, cs.ExtFunc)
	if err != nil {
		return err
	}

	sort.Sort(valueSet(dbSetArray))

	dupCheck := map[string][]int{}
	for i := range extSetArray {
		_, ok := dupCheck[extSetArray[i].key]
		if !ok {
			dupCheck[extSetArray[i].key] = []int{}
		}
		dupCheck[extSetArray[i].key] = append(dupCheck[extSetArray[i].key], i)
	}

	var dupValue reflect.Value
	storeDup := false
	if duplicated != nil {
		storeDup = true
		dupValue = reflect.Indirect(reflect.ValueOf(duplicated))
	}

	errs := make([]error, 0)
	newExtSetArray := make([]valueElement, 0)
	duplicateMap := map[string]bool{}
	for k, idx := range dupCheck {
		if len(idx) > 1 {
			if !storeDup {
				log.Warningf("CompareSets Duplicate ID: %s (%d)", k, len(idx))
				errs = append(errs, errors.Wrapf(errors.ErrDuplicateId, "duplicated id: %s (%d)", k, len(idx)))
			} else {
				// store in dupValue
				dupArrays := reflect.MakeSlice(reflect.SliceOf(extSetArray[idx[0]].value.Type()), len(idx), len(idx))
				for i := 0; i < len(idx); i++ {
					dupArrays.Index(i).Set(extSetArray[idx[i]].value)
				}
				dupValue.SetMapIndex(reflect.ValueOf(k), dupArrays)
				duplicateMap[k] = true
			}
		} else {
			newExtSetArray = append(newExtSetArray, extSetArray[idx[0]])
		}
	}

	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}

	extSetArray = newExtSetArray

	sort.Sort(valueSet(extSetArray))

	removedValue := reflect.Indirect(reflect.ValueOf(removed))
	commonDBValue := reflect.Indirect(reflect.ValueOf(commonDB))
	commonExtValue := reflect.Indirect(reflect.ValueOf(commonExt))
	addedValue := reflect.Indirect(reflect.ValueOf(added))

	i := 0
	j := 0
	for i < len(dbSetArray) || j < len(extSetArray) {
		if i < len(dbSetArray) && j < len(extSetArray) {
			cmp := strings.Compare(dbSetArray[i].key, extSetArray[j].key)
			if cmp == 0 {
				newVal1 := reflect.Append(commonDBValue, dbSetArray[i].value)
				commonDBValue.Set(newVal1)
				newVal2 := reflect.Append(commonExtValue, extSetArray[j].value)
				commonExtValue.Set(newVal2)
				i += 1
				j += 1
			} else if cmp < 0 {
				if _, ok := duplicateMap[dbSetArray[i].key]; !ok && len(dbSetArray[i].key) > 0 {
					newVal := reflect.Append(removedValue, dbSetArray[i].value)
					removedValue.Set(newVal)
				}
				i += 1
			} else {
				newVal := reflect.Append(addedValue, extSetArray[j].value)
				addedValue.Set(newVal)
				j += 1
			}
		} else if i >= len(dbSetArray) {
			newVal := reflect.Append(addedValue, extSetArray[j].value)
			addedValue.Set(newVal)
			j += 1
		} else if j >= len(extSetArray) {
			if _, ok := duplicateMap[dbSetArray[i].key]; !ok && len(dbSetArray[i].key) > 0 {
				newVal := reflect.Append(removedValue, dbSetArray[i].value)
				removedValue.Set(newVal)
			}
			i += 1
		}
	}
	return nil
}

func CompareSets(dbSet interface{}, extSet interface{}, removed interface{}, commonDB interface{}, commonExt interface{}, added interface{}) error {
	return CompareSets2(dbSet, extSet, removed, commonDB, commonExt, added, nil)
}

func CompareSets2(dbSet interface{}, extSet interface{}, removed interface{}, commonDB interface{}, commonExt interface{}, added interface{}, duplicated interface{}) error {
	return CompareSetsFunc(SCompareSet{
		DBFunc:  "GetExternalId",
		DBSet:   dbSet,
		ExtFunc: "GetGlobalId",
		ExtSet:  extSet,
	}, removed, commonDB, commonExt, added, duplicated)
}
