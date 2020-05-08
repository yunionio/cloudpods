// DO NOT EDIT: automatically generated code

package types

import (
	"fmt"
	"strings"
)

func OvsdbCmdArgString(a string) string {
	return fmt.Sprintf("%q", a)
}

func OvsdbCmdArgsString(field string, a string) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvsdbCmdArgString(a))}
}

func OvsdbCmdArgsStringOptional(field string, a *string) []string {
	if a == nil {
		return nil
	}
	return OvsdbCmdArgsString(field, *a)
}

func OvsdbCmdArgsStringMultiples(field string, a []string) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvsdbCmdArgString(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvsdbCmdArgsMapStringString(field string, a map[string]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgString(aK), OvsdbCmdArgString(aV)))
	}
	return r
}

func OvsdbCmdArgsMapStringInteger(field string, a map[string]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgString(aK), OvsdbCmdArgInteger(aV)))
	}
	return r
}

func OvsdbCmdArgsMapStringReal(field string, a map[string]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgString(aK), OvsdbCmdArgReal(aV)))
	}
	return r
}

func OvsdbCmdArgsMapStringBoolean(field string, a map[string]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgString(aK), OvsdbCmdArgBoolean(aV)))
	}
	return r
}

func OvsdbCmdArgsMapStringUuid(field string, a map[string]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgString(aK), OvsdbCmdArgUuid(aV)))
	}
	return r
}

func OvsdbCmdArgInteger(a int64) string {
	return fmt.Sprintf("%d", a)
}

func OvsdbCmdArgsInteger(field string, a int64) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvsdbCmdArgInteger(a))}
}

func OvsdbCmdArgsIntegerOptional(field string, a *int64) []string {
	if a == nil {
		return nil
	}
	return OvsdbCmdArgsInteger(field, *a)
}

func OvsdbCmdArgsIntegerMultiples(field string, a []int64) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvsdbCmdArgInteger(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvsdbCmdArgsMapIntegerString(field string, a map[int64]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgInteger(aK), OvsdbCmdArgString(aV)))
	}
	return r
}

func OvsdbCmdArgsMapIntegerInteger(field string, a map[int64]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgInteger(aK), OvsdbCmdArgInteger(aV)))
	}
	return r
}

func OvsdbCmdArgsMapIntegerReal(field string, a map[int64]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgInteger(aK), OvsdbCmdArgReal(aV)))
	}
	return r
}

func OvsdbCmdArgsMapIntegerBoolean(field string, a map[int64]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgInteger(aK), OvsdbCmdArgBoolean(aV)))
	}
	return r
}

func OvsdbCmdArgsMapIntegerUuid(field string, a map[int64]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgInteger(aK), OvsdbCmdArgUuid(aV)))
	}
	return r
}

func OvsdbCmdArgReal(a float64) string {
	return fmt.Sprintf("%f", a)
}

func OvsdbCmdArgsReal(field string, a float64) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvsdbCmdArgReal(a))}
}

func OvsdbCmdArgsRealOptional(field string, a *float64) []string {
	if a == nil {
		return nil
	}
	return OvsdbCmdArgsReal(field, *a)
}

func OvsdbCmdArgsRealMultiples(field string, a []float64) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvsdbCmdArgReal(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvsdbCmdArgsMapRealString(field string, a map[float64]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgReal(aK), OvsdbCmdArgString(aV)))
	}
	return r
}

func OvsdbCmdArgsMapRealInteger(field string, a map[float64]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgReal(aK), OvsdbCmdArgInteger(aV)))
	}
	return r
}

func OvsdbCmdArgsMapRealReal(field string, a map[float64]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgReal(aK), OvsdbCmdArgReal(aV)))
	}
	return r
}

func OvsdbCmdArgsMapRealBoolean(field string, a map[float64]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgReal(aK), OvsdbCmdArgBoolean(aV)))
	}
	return r
}

func OvsdbCmdArgsMapRealUuid(field string, a map[float64]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgReal(aK), OvsdbCmdArgUuid(aV)))
	}
	return r
}

func OvsdbCmdArgBoolean(a bool) string {
	return fmt.Sprintf("%v", a)
}

func OvsdbCmdArgsBoolean(field string, a bool) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvsdbCmdArgBoolean(a))}
}

func OvsdbCmdArgsBooleanOptional(field string, a *bool) []string {
	if a == nil {
		return nil
	}
	return OvsdbCmdArgsBoolean(field, *a)
}

func OvsdbCmdArgsBooleanMultiples(field string, a []bool) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvsdbCmdArgBoolean(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvsdbCmdArgsMapBooleanString(field string, a map[bool]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgBoolean(aK), OvsdbCmdArgString(aV)))
	}
	return r
}

func OvsdbCmdArgsMapBooleanInteger(field string, a map[bool]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgBoolean(aK), OvsdbCmdArgInteger(aV)))
	}
	return r
}

func OvsdbCmdArgsMapBooleanReal(field string, a map[bool]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgBoolean(aK), OvsdbCmdArgReal(aV)))
	}
	return r
}

func OvsdbCmdArgsMapBooleanBoolean(field string, a map[bool]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgBoolean(aK), OvsdbCmdArgBoolean(aV)))
	}
	return r
}

func OvsdbCmdArgsMapBooleanUuid(field string, a map[bool]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgBoolean(aK), OvsdbCmdArgUuid(aV)))
	}
	return r
}

func OvsdbCmdArgUuid(a string) string {
	return fmt.Sprintf("%s", a)
}

func OvsdbCmdArgsUuid(field string, a string) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvsdbCmdArgUuid(a))}
}

func OvsdbCmdArgsUuidOptional(field string, a *string) []string {
	if a == nil {
		return nil
	}
	return OvsdbCmdArgsUuid(field, *a)
}

func OvsdbCmdArgsUuidMultiples(field string, a []string) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvsdbCmdArgUuid(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvsdbCmdArgsMapUuidString(field string, a map[string]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgUuid(aK), OvsdbCmdArgString(aV)))
	}
	return r
}

func OvsdbCmdArgsMapUuidInteger(field string, a map[string]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgUuid(aK), OvsdbCmdArgInteger(aV)))
	}
	return r
}

func OvsdbCmdArgsMapUuidReal(field string, a map[string]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgUuid(aK), OvsdbCmdArgReal(aV)))
	}
	return r
}

func OvsdbCmdArgsMapUuidBoolean(field string, a map[string]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgUuid(aK), OvsdbCmdArgBoolean(aV)))
	}
	return r
}

func OvsdbCmdArgsMapUuidUuid(field string, a map[string]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvsdbCmdArgUuid(aK), OvsdbCmdArgUuid(aV)))
	}
	return r
}
