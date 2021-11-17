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

package sqlchemy

import (
	"reflect"

	"yunion.io/x/log"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"
)

// Fetch method fetches the values of a struct whose primary key values have been set
// input is a pointer to the model to be populated
func (ts *STableSpec) Fetch(dt interface{}) error {
	q := ts.Query()
	dataValue := reflect.ValueOf(dt).Elem()
	fields := reflectutils.FetchStructFieldValueSet(dataValue)
	for _, c := range ts.columns {
		priVal, _ := fields.GetInterface(c.Name())
		if c.IsPrimary() && !gotypes.IsNil(priVal) { // skip update primary key
			q = q.Equals(c.Name(), priVal)
		}
	}
	return q.First(dt)
}

// FetchAll method fetches the values of an array of structs whose primary key values have been set
// input is a pointer to the array of models to be populated
func (ts *STableSpec) FetchAll(dest interface{}) error {
	arrayType := reflect.TypeOf(dest).Elem()
	if arrayType.Kind() != reflect.Array && arrayType.Kind() != reflect.Slice {
		return errors.Wrap(ErrNeedsArray, "dest is not an array or slice")
	}

	arrayValue := reflect.ValueOf(dest).Elem()

	primaryCols := ts.PrimaryColumns()
	if len(primaryCols) != 1 {
		return errors.Wrap(ErrNotSupported, "support 1 primary key only")
	}
	primaryCol := primaryCols[0]

	keyValues := make([]interface{}, arrayValue.Len())
	for i := 0; i < arrayValue.Len(); i++ {
		eleValue := arrayValue.Index(i)
		fields := reflectutils.FetchStructFieldValueSet(eleValue)
		keyValues[i], _ = fields.GetInterface(primaryCol.Name())
	}
	q := ts.Query().In(primaryCol.Name(), keyValues)

	tmpDestMaps, err := q.AllStringMap()
	if err != nil {
		return errors.Wrap(err, "q.AllStringMap")
	}

	tmpDestMapMap := make(map[string]map[string]string)
	for i := 0; i < len(tmpDestMaps); i++ {
		tmpDestMapMap[tmpDestMaps[i][primaryCol.Name()]] = tmpDestMaps[i]
	}

	for i := 0; i < arrayValue.Len(); i++ {
		keyValueStr := getStringValue(keyValues[i])
		if tmpMap, ok := tmpDestMapMap[keyValueStr]; ok {
			err = mapString2Struct(tmpMap, arrayValue.Index(i))
			if err != nil {
				return errors.Wrapf(err, "mapString2Struct %d:%s", i, keyValueStr)
			}
		} else {
			log.Warningf("element %d:%s not found in fetch result", i, keyValueStr)
		}
	}

	return nil
}
