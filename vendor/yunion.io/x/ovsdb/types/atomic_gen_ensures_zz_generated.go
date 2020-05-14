// DO NOT EDIT: automatically generated code

package types

import "github.com/pkg/errors"

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

func EnsureUuid(val interface{}) string {
	val = ensureTyped(val, "uuid")
	r, ok := val.(string)
	if !ok {
		panicErr("bad uuid value")
	}
	return r
}

func EnsureUuidMultiples(val interface{}) []string {
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
			r[i] = EnsureUuid(val)
		}
		return r
	}
	panic("uuid multiple: unexpected type: " + typ)
}

func EnsureString(val interface{}) string {
	if r, ok := val.(string); ok {
		return string(r)
	}
	panic(ErrBadType)
}

func EnsureStringMultiples(val interface{}) []string {
	if ok := probeEmpty(val); ok {
		return nil
	}
	if r, ok := val.(string); ok {
		return []string{string(r)}
	}
	mulVal := ensureMultiples(val, "set")
	if len(mulVal) == 0 {
		return nil
	}
	r := make([]string, len(mulVal))
	for i, val := range mulVal {
		r[i] = EnsureString(val)
	}
	return r
}

func EnsureStringOptional(val interface{}) *string {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := EnsureString(val)
	return &r
}

func EnsureMapStringString(val interface{}) map[string]string {
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
		k := EnsureString(pair[0])
		v := EnsureString(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapStringInteger(val interface{}) map[string]int64 {
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
		k := EnsureString(pair[0])
		v := EnsureInteger(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapStringReal(val interface{}) map[string]float64 {
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
		k := EnsureString(pair[0])
		v := EnsureReal(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapStringBoolean(val interface{}) map[string]bool {
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
		k := EnsureString(pair[0])
		v := EnsureBoolean(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapStringUuid(val interface{}) map[string]string {
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
		k := EnsureString(pair[0])
		v := EnsureUuid(pair[1])
		r[k] = v
	}
	return r
}

func EnsureInteger(val interface{}) int64 {
	if r, ok := val.(float64); ok {
		return int64(r)
	}
	panic(ErrBadType)
}

func EnsureIntegerMultiples(val interface{}) []int64 {
	if ok := probeEmpty(val); ok {
		return nil
	}
	if r, ok := val.(float64); ok {
		return []int64{int64(r)}
	}
	mulVal := ensureMultiples(val, "set")
	if len(mulVal) == 0 {
		return nil
	}
	r := make([]int64, len(mulVal))
	for i, val := range mulVal {
		r[i] = EnsureInteger(val)
	}
	return r
}

func EnsureIntegerOptional(val interface{}) *int64 {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := EnsureInteger(val)
	return &r
}

func EnsureMapIntegerString(val interface{}) map[int64]string {
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
		k := EnsureInteger(pair[0])
		v := EnsureString(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapIntegerInteger(val interface{}) map[int64]int64 {
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
		k := EnsureInteger(pair[0])
		v := EnsureInteger(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapIntegerReal(val interface{}) map[int64]float64 {
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
		k := EnsureInteger(pair[0])
		v := EnsureReal(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapIntegerBoolean(val interface{}) map[int64]bool {
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
		k := EnsureInteger(pair[0])
		v := EnsureBoolean(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapIntegerUuid(val interface{}) map[int64]string {
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
		k := EnsureInteger(pair[0])
		v := EnsureUuid(pair[1])
		r[k] = v
	}
	return r
}

func EnsureReal(val interface{}) float64 {
	if r, ok := val.(float64); ok {
		return float64(r)
	}
	panic(ErrBadType)
}

func EnsureRealMultiples(val interface{}) []float64 {
	if ok := probeEmpty(val); ok {
		return nil
	}
	if r, ok := val.(float64); ok {
		return []float64{float64(r)}
	}
	mulVal := ensureMultiples(val, "set")
	if len(mulVal) == 0 {
		return nil
	}
	r := make([]float64, len(mulVal))
	for i, val := range mulVal {
		r[i] = EnsureReal(val)
	}
	return r
}

func EnsureRealOptional(val interface{}) *float64 {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := EnsureReal(val)
	return &r
}

func EnsureMapRealString(val interface{}) map[float64]string {
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
		k := EnsureReal(pair[0])
		v := EnsureString(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapRealInteger(val interface{}) map[float64]int64 {
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
		k := EnsureReal(pair[0])
		v := EnsureInteger(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapRealReal(val interface{}) map[float64]float64 {
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
		k := EnsureReal(pair[0])
		v := EnsureReal(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapRealBoolean(val interface{}) map[float64]bool {
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
		k := EnsureReal(pair[0])
		v := EnsureBoolean(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapRealUuid(val interface{}) map[float64]string {
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
		k := EnsureReal(pair[0])
		v := EnsureUuid(pair[1])
		r[k] = v
	}
	return r
}

func EnsureBoolean(val interface{}) bool {
	if r, ok := val.(bool); ok {
		return bool(r)
	}
	panic(ErrBadType)
}

func EnsureBooleanMultiples(val interface{}) []bool {
	if ok := probeEmpty(val); ok {
		return nil
	}
	if r, ok := val.(bool); ok {
		return []bool{bool(r)}
	}
	mulVal := ensureMultiples(val, "set")
	if len(mulVal) == 0 {
		return nil
	}
	r := make([]bool, len(mulVal))
	for i, val := range mulVal {
		r[i] = EnsureBoolean(val)
	}
	return r
}

func EnsureBooleanOptional(val interface{}) *bool {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := EnsureBoolean(val)
	return &r
}

func EnsureMapBooleanString(val interface{}) map[bool]string {
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
		k := EnsureBoolean(pair[0])
		v := EnsureString(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapBooleanInteger(val interface{}) map[bool]int64 {
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
		k := EnsureBoolean(pair[0])
		v := EnsureInteger(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapBooleanReal(val interface{}) map[bool]float64 {
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
		k := EnsureBoolean(pair[0])
		v := EnsureReal(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapBooleanBoolean(val interface{}) map[bool]bool {
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
		k := EnsureBoolean(pair[0])
		v := EnsureBoolean(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapBooleanUuid(val interface{}) map[bool]string {
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
		k := EnsureBoolean(pair[0])
		v := EnsureUuid(pair[1])
		r[k] = v
	}
	return r
}

func EnsureUuidOptional(val interface{}) *string {
	if ok := probeEmpty(val); ok {
		return nil
	}
	r := EnsureUuid(val)
	return &r
}

func EnsureMapUuidString(val interface{}) map[string]string {
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
		k := EnsureUuid(pair[0])
		v := EnsureString(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapUuidInteger(val interface{}) map[string]int64 {
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
		k := EnsureUuid(pair[0])
		v := EnsureInteger(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapUuidReal(val interface{}) map[string]float64 {
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
		k := EnsureUuid(pair[0])
		v := EnsureReal(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapUuidBoolean(val interface{}) map[string]bool {
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
		k := EnsureUuid(pair[0])
		v := EnsureBoolean(pair[1])
		r[k] = v
	}
	return r
}

func EnsureMapUuidUuid(val interface{}) map[string]string {
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
		k := EnsureUuid(pair[0])
		v := EnsureUuid(pair[1])
		r[k] = v
	}
	return r
}
