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

package ovnutil

import (
	"yunion.io/x/pkg/errors"
)

const (
	ErrBadType = errors.Error("bad type")
)

func panicErr(msg string) {
	panic(errors.Wrap(ErrBadType, msg))
}

func panicErrf(fmtStr string, s ...interface{}) {
	panic(errors.Wrapf(ErrBadType, fmtStr, s...))
}

func ensureTypedPair(val interface{}) (string, interface{}) {
	arr, ok := val.([]interface{})
	if !ok {
		panicErr("ensureTypedPair: not an array")
	}
	if len(arr) != 2 {
		panicErrf("ensureTypedPair: length is %d, want 2", len(arr))
	}
	typ, ok := arr[0].(string)
	if !ok {
		panicErr("ensureTypedPair: type not a string")
	}
	return typ, arr[1]
}

func ensureTyped(val interface{}, typ string) interface{} {
	gotTyp, r := ensureTypedPair(val)
	if gotTyp != typ {
		panicErrf("ensureMultiples: got %s, want %s", gotTyp, typ)
	}
	return r
}

func ensureMultiples(val interface{}, typ string) []interface{} {
	val = ensureTyped(val, typ)
	mulVal, ok := val.([]interface{})
	if !ok {
		panicErr("ensureMultiples: val is not an array")
	}
	return mulVal
}

func probeEmpty(val interface{}) (r bool) {
	defer func() {
		recover()
	}()
	empty := ensureMultiples(val, "set")
	if len(empty) != 0 {
		return false
	}
	return true
}

func ensureUuid(val interface{}) string {
	val = ensureTyped(val, "uuid")
	r, ok := val.(string)
	if !ok {
		panicErr("bad uuid value")
	}
	return r
}

func ensureUuidMultiples(val interface{}) []string {
	typ, val1 := ensureTypedPair(val)
	if typ == "uuid" {
		r, ok := val1.(string)
		if !ok {
			panicErr("uuid multiples: expect a string")
		}
		return []string{r}
	}
	if typ == "set" {
		mulVal, ok := val1.([]interface{})
		if !ok {
			panicErr("uuid multiples: expect an array")
		}
		if len(mulVal) == 0 {
			return nil
		}
		r := make([]string, len(mulVal))
		for i, val := range mulVal {
			r[i] = ensureUuid(val)
		}
		return r
	}
	panic("uuid multiple: unexpected type: " + typ)
}

func ensureBoolean(val interface{}) bool {
	if r, ok := val.(bool); ok {
		return r
	}
	panic(ErrBadType)
}

func ensureBooleanMultiples(val interface{}) []bool {
	if ok := probeEmpty(val); ok {
		return nil
	}
	if r, ok := val.(bool); ok {
		return []bool{r}
	}
	mulVal := ensureMultiples(val, "set")
	if len(mulVal) == 0 {
		return nil
	}
	r := make([]bool, len(mulVal))
	for i, val := range mulVal {
		r[i] = ensureBoolean(val)
	}
	return r
}

func ensureBooleanOptional(val interface{}) *bool {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := ensureBoolean(val)
	return &r
}

func ensureMapBooleanUuid(val interface{}) map[bool]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[bool]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureBoolean(pair[0])
		v := ensureUuid(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapBooleanString(val interface{}) map[bool]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[bool]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureBoolean(pair[0])
		v := ensureString(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapBooleanInteger(val interface{}) map[bool]int64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[bool]int64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureBoolean(pair[0])
		v := ensureInteger(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapBooleanBoolean(val interface{}) map[bool]bool {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[bool]bool{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureBoolean(pair[0])
		v := ensureBoolean(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapBooleanReal(val interface{}) map[bool]float64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[bool]float64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureBoolean(pair[0])
		v := ensureReal(pair[1])
		r[k] = v
	}
	return r
}

func ensureReal(val interface{}) float64 {
	if r, ok := val.(float64); ok {
		return r
	}
	panic(ErrBadType)
}

func ensureRealMultiples(val interface{}) []float64 {
	if ok := probeEmpty(val); ok {
		return nil
	}
	if r, ok := val.(float64); ok {
		return []float64{r}
	}
	mulVal := ensureMultiples(val, "set")
	if len(mulVal) == 0 {
		return nil
	}
	r := make([]float64, len(mulVal))
	for i, val := range mulVal {
		r[i] = ensureReal(val)
	}
	return r
}

func ensureRealOptional(val interface{}) *float64 {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := ensureReal(val)
	return &r
}

func ensureMapRealString(val interface{}) map[float64]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[float64]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureReal(pair[0])
		v := ensureString(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapRealInteger(val interface{}) map[float64]int64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[float64]int64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureReal(pair[0])
		v := ensureInteger(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapRealBoolean(val interface{}) map[float64]bool {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[float64]bool{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureReal(pair[0])
		v := ensureBoolean(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapRealReal(val interface{}) map[float64]float64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[float64]float64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureReal(pair[0])
		v := ensureReal(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapRealUuid(val interface{}) map[float64]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[float64]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureReal(pair[0])
		v := ensureUuid(pair[1])
		r[k] = v
	}
	return r
}

func ensureUuidOptional(val interface{}) *string {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := ensureUuid(val)
	return &r
}

func ensureMapUuidReal(val interface{}) map[string]float64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]float64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureUuid(pair[0])
		v := ensureReal(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapUuidUuid(val interface{}) map[string]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureUuid(pair[0])
		v := ensureUuid(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapUuidString(val interface{}) map[string]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureUuid(pair[0])
		v := ensureString(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapUuidInteger(val interface{}) map[string]int64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]int64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureUuid(pair[0])
		v := ensureInteger(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapUuidBoolean(val interface{}) map[string]bool {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]bool{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureUuid(pair[0])
		v := ensureBoolean(pair[1])
		r[k] = v
	}
	return r
}

func ensureString(val interface{}) string {
	if r, ok := val.(string); ok {
		return r
	}
	panic(ErrBadType)
}

func ensureStringMultiples(val interface{}) []string {
	if ok := probeEmpty(val); ok {
		return nil
	}
	if r, ok := val.(string); ok {
		return []string{r}
	}
	mulVal := ensureMultiples(val, "set")
	if len(mulVal) == 0 {
		return nil
	}
	r := make([]string, len(mulVal))
	for i, val := range mulVal {
		r[i] = ensureString(val)
	}
	return r
}

func ensureStringOptional(val interface{}) *string {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := ensureString(val)
	return &r
}

func ensureMapStringUuid(val interface{}) map[string]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureString(pair[0])
		v := ensureUuid(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapStringString(val interface{}) map[string]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureString(pair[0])
		v := ensureString(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapStringInteger(val interface{}) map[string]int64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]int64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureString(pair[0])
		v := ensureInteger(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapStringBoolean(val interface{}) map[string]bool {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]bool{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureString(pair[0])
		v := ensureBoolean(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapStringReal(val interface{}) map[string]float64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[string]float64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureString(pair[0])
		v := ensureReal(pair[1])
		r[k] = v
	}
	return r
}

func ensureInteger(val interface{}) int64 {
	if r, ok := val.(int64); ok {
		return r
	}
	panic(ErrBadType)
}

func ensureIntegerMultiples(val interface{}) []int64 {
	if ok := probeEmpty(val); ok {
		return nil
	}
	if r, ok := val.(int64); ok {
		return []int64{r}
	}
	mulVal := ensureMultiples(val, "set")
	if len(mulVal) == 0 {
		return nil
	}
	r := make([]int64, len(mulVal))
	for i, val := range mulVal {
		r[i] = ensureInteger(val)
	}
	return r
}

func ensureIntegerOptional(val interface{}) *int64 {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := ensureInteger(val)
	return &r
}

func ensureMapIntegerReal(val interface{}) map[int64]float64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[int64]float64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureInteger(pair[0])
		v := ensureReal(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapIntegerUuid(val interface{}) map[int64]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[int64]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureInteger(pair[0])
		v := ensureUuid(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapIntegerString(val interface{}) map[int64]string {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[int64]string{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureInteger(pair[0])
		v := ensureString(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapIntegerInteger(val interface{}) map[int64]int64 {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[int64]int64{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureInteger(pair[0])
		v := ensureInteger(pair[1])
		r[k] = v
	}
	return r
}

func ensureMapIntegerBoolean(val interface{}) map[int64]bool {
	mulVal := ensureMultiples(val, "map")
	if len(mulVal) == 0 {
		return nil
	}
	r := map[int64]bool{}
	for _, pairVal := range mulVal {
		pair, ok := pairVal.([]interface{})
		if !ok {
			panicErr("map: not an array")
		}
		if len(pair) != 2 {
			panicErr("map: not a pair")
		}
		k := ensureInteger(pair[0])
		v := ensureBoolean(pair[1])
		r[k] = v
	}
	return r
}
