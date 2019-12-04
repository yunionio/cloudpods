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
	"fmt"
	"strings"
)

func OvnArgUuid(a string) string {
	return fmt.Sprintf("%s", a)
}
func OvnArgsUuid(field string, a string) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvnArgUuid(a))}
}

func OvnArgsUuidOptional(field string, a *string) []string {
	if a == nil {
		return nil
	}
	return OvnArgsUuid(field, *a)
}

func OvnArgsUuidMultiples(field string, a []string) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvnArgUuid(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvnArgsMapUuidReal(field string, a map[string]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgUuid(aK), OvnArgReal(aV)))
	}
	return r
}

func OvnArgsMapUuidUuid(field string, a map[string]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgUuid(aK), OvnArgUuid(aV)))
	}
	return r
}

func OvnArgsMapUuidString(field string, a map[string]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgUuid(aK), OvnArgString(aV)))
	}
	return r
}

func OvnArgsMapUuidInteger(field string, a map[string]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgUuid(aK), OvnArgInteger(aV)))
	}
	return r
}

func OvnArgsMapUuidBoolean(field string, a map[string]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgUuid(aK), OvnArgBoolean(aV)))
	}
	return r
}

func OvnArgString(a string) string {
	return fmt.Sprintf("%q", a)
}
func OvnArgsString(field string, a string) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvnArgString(a))}
}

func OvnArgsStringOptional(field string, a *string) []string {
	if a == nil {
		return nil
	}
	return OvnArgsString(field, *a)
}

func OvnArgsStringMultiples(field string, a []string) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvnArgString(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvnArgsMapStringUuid(field string, a map[string]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgString(aK), OvnArgUuid(aV)))
	}
	return r
}

func OvnArgsMapStringString(field string, a map[string]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgString(aK), OvnArgString(aV)))
	}
	return r
}

func OvnArgsMapStringInteger(field string, a map[string]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgString(aK), OvnArgInteger(aV)))
	}
	return r
}

func OvnArgsMapStringBoolean(field string, a map[string]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgString(aK), OvnArgBoolean(aV)))
	}
	return r
}

func OvnArgsMapStringReal(field string, a map[string]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgString(aK), OvnArgReal(aV)))
	}
	return r
}

func OvnArgInteger(a int64) string {
	return fmt.Sprintf("%d", a)
}
func OvnArgsInteger(field string, a int64) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvnArgInteger(a))}
}

func OvnArgsIntegerOptional(field string, a *int64) []string {
	if a == nil {
		return nil
	}
	return OvnArgsInteger(field, *a)
}

func OvnArgsIntegerMultiples(field string, a []int64) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvnArgInteger(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvnArgsMapIntegerUuid(field string, a map[int64]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgInteger(aK), OvnArgUuid(aV)))
	}
	return r
}

func OvnArgsMapIntegerString(field string, a map[int64]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgInteger(aK), OvnArgString(aV)))
	}
	return r
}

func OvnArgsMapIntegerInteger(field string, a map[int64]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgInteger(aK), OvnArgInteger(aV)))
	}
	return r
}

func OvnArgsMapIntegerBoolean(field string, a map[int64]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgInteger(aK), OvnArgBoolean(aV)))
	}
	return r
}

func OvnArgsMapIntegerReal(field string, a map[int64]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgInteger(aK), OvnArgReal(aV)))
	}
	return r
}

func OvnArgBoolean(a bool) string {
	return fmt.Sprintf("%v", a)
}
func OvnArgsBoolean(field string, a bool) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvnArgBoolean(a))}
}

func OvnArgsBooleanOptional(field string, a *bool) []string {
	if a == nil {
		return nil
	}
	return OvnArgsBoolean(field, *a)
}

func OvnArgsBooleanMultiples(field string, a []bool) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvnArgBoolean(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvnArgsMapBooleanUuid(field string, a map[bool]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgBoolean(aK), OvnArgUuid(aV)))
	}
	return r
}

func OvnArgsMapBooleanString(field string, a map[bool]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgBoolean(aK), OvnArgString(aV)))
	}
	return r
}

func OvnArgsMapBooleanInteger(field string, a map[bool]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgBoolean(aK), OvnArgInteger(aV)))
	}
	return r
}

func OvnArgsMapBooleanBoolean(field string, a map[bool]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgBoolean(aK), OvnArgBoolean(aV)))
	}
	return r
}

func OvnArgsMapBooleanReal(field string, a map[bool]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgBoolean(aK), OvnArgReal(aV)))
	}
	return r
}

func OvnArgReal(a float64) string {
	return fmt.Sprintf("%f", a)
}
func OvnArgsReal(field string, a float64) []string {
	return []string{fmt.Sprintf("%s=%s", field, OvnArgReal(a))}
}

func OvnArgsRealOptional(field string, a *float64) []string {
	if a == nil {
		return nil
	}
	return OvnArgsReal(field, *a)
}

func OvnArgsRealMultiples(field string, a []float64) []string {
	if len(a) == 0 {
		return nil
	}
	elArgs := make([]string, len(a))
	for i, el := range a {
		elArgs[i] = OvnArgReal(el)
	}
	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))
	return []string{arg}
}

func OvnArgsMapRealReal(field string, a map[float64]float64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgReal(aK), OvnArgReal(aV)))
	}
	return r
}

func OvnArgsMapRealUuid(field string, a map[float64]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgReal(aK), OvnArgUuid(aV)))
	}
	return r
}

func OvnArgsMapRealString(field string, a map[float64]string) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgReal(aK), OvnArgString(aV)))
	}
	return r
}

func OvnArgsMapRealInteger(field string, a map[float64]int64) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgReal(aK), OvnArgInteger(aV)))
	}
	return r
}

func OvnArgsMapRealBoolean(field string, a map[float64]bool) []string {
	if len(a) == 0 {
		return nil
	}
	r := make([]string, 0, len(a))
	for aK, aV := range a {
		r = append(r, fmt.Sprintf("%s:%s=%s", field, OvnArgReal(aK), OvnArgBoolean(aV)))
	}
	return r
}
