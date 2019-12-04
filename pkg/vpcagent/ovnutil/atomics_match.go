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

func matchIntegerIfNonZero(a, b int64) bool {
	var z int64
	if b == z {
		return true
	}
	return matchInteger(a, b)
}

func matchInteger(a, b int64) bool {
	return a == b
}

func matchIntegerOptionalIfNonZero(a, b *int64) bool {
	if b == nil {
		return true
	}
	return matchIntegerOptional(a, b)
}

func matchIntegerOptional(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func matchIntegerMultiplesIfNonZero(a, b []int64) bool {
	if b == nil {
		return true
	}
	return matchIntegerMultiples(a, b)
}

func matchIntegerMultiples(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := make([]int64, len(b))
	copy(bCopy, b)
	for _, elA := range a {
		for i := len(bCopy) - 1; i >= 0; i-- {
			elB := bCopy[i]
			if elA == elB {
				bCopy = append(bCopy[:i], bCopy[i+1:]...)
			}
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapIntegerUuidIfNonZero(a, b map[int64]string) bool {
	if b == nil {
		return true
	}
	return matchMapIntegerUuid(a, b)
}

func matchMapIntegerUuid(a, b map[int64]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[int64]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapIntegerStringIfNonZero(a, b map[int64]string) bool {
	if b == nil {
		return true
	}
	return matchMapIntegerString(a, b)
}

func matchMapIntegerString(a, b map[int64]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[int64]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapIntegerIntegerIfNonZero(a, b map[int64]int64) bool {
	if b == nil {
		return true
	}
	return matchMapIntegerInteger(a, b)
}

func matchMapIntegerInteger(a, b map[int64]int64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[int64]int64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapIntegerBooleanIfNonZero(a, b map[int64]bool) bool {
	if b == nil {
		return true
	}
	return matchMapIntegerBoolean(a, b)
}

func matchMapIntegerBoolean(a, b map[int64]bool) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[int64]bool{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapIntegerRealIfNonZero(a, b map[int64]float64) bool {
	if b == nil {
		return true
	}
	return matchMapIntegerReal(a, b)
}

func matchMapIntegerReal(a, b map[int64]float64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[int64]float64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchBooleanIfNonZero(a, b bool) bool {
	var z bool
	if b == z {
		return true
	}
	return matchBoolean(a, b)
}

func matchBoolean(a, b bool) bool {
	return a == b
}

func matchBooleanOptionalIfNonZero(a, b *bool) bool {
	if b == nil {
		return true
	}
	return matchBooleanOptional(a, b)
}

func matchBooleanOptional(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func matchBooleanMultiplesIfNonZero(a, b []bool) bool {
	if b == nil {
		return true
	}
	return matchBooleanMultiples(a, b)
}

func matchBooleanMultiples(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := make([]bool, len(b))
	copy(bCopy, b)
	for _, elA := range a {
		for i := len(bCopy) - 1; i >= 0; i-- {
			elB := bCopy[i]
			if elA == elB {
				bCopy = append(bCopy[:i], bCopy[i+1:]...)
			}
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapBooleanBooleanIfNonZero(a, b map[bool]bool) bool {
	if b == nil {
		return true
	}
	return matchMapBooleanBoolean(a, b)
}

func matchMapBooleanBoolean(a, b map[bool]bool) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[bool]bool{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapBooleanRealIfNonZero(a, b map[bool]float64) bool {
	if b == nil {
		return true
	}
	return matchMapBooleanReal(a, b)
}

func matchMapBooleanReal(a, b map[bool]float64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[bool]float64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapBooleanUuidIfNonZero(a, b map[bool]string) bool {
	if b == nil {
		return true
	}
	return matchMapBooleanUuid(a, b)
}

func matchMapBooleanUuid(a, b map[bool]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[bool]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapBooleanStringIfNonZero(a, b map[bool]string) bool {
	if b == nil {
		return true
	}
	return matchMapBooleanString(a, b)
}

func matchMapBooleanString(a, b map[bool]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[bool]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapBooleanIntegerIfNonZero(a, b map[bool]int64) bool {
	if b == nil {
		return true
	}
	return matchMapBooleanInteger(a, b)
}

func matchMapBooleanInteger(a, b map[bool]int64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[bool]int64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchRealIfNonZero(a, b float64) bool {
	var z float64
	if b == z {
		return true
	}
	return matchReal(a, b)
}

func matchReal(a, b float64) bool {
	return a == b
}

func matchRealOptionalIfNonZero(a, b *float64) bool {
	if b == nil {
		return true
	}
	return matchRealOptional(a, b)
}

func matchRealOptional(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func matchRealMultiplesIfNonZero(a, b []float64) bool {
	if b == nil {
		return true
	}
	return matchRealMultiples(a, b)
}

func matchRealMultiples(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := make([]float64, len(b))
	copy(bCopy, b)
	for _, elA := range a {
		for i := len(bCopy) - 1; i >= 0; i-- {
			elB := bCopy[i]
			if elA == elB {
				bCopy = append(bCopy[:i], bCopy[i+1:]...)
			}
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapRealUuidIfNonZero(a, b map[float64]string) bool {
	if b == nil {
		return true
	}
	return matchMapRealUuid(a, b)
}

func matchMapRealUuid(a, b map[float64]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[float64]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapRealStringIfNonZero(a, b map[float64]string) bool {
	if b == nil {
		return true
	}
	return matchMapRealString(a, b)
}

func matchMapRealString(a, b map[float64]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[float64]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapRealIntegerIfNonZero(a, b map[float64]int64) bool {
	if b == nil {
		return true
	}
	return matchMapRealInteger(a, b)
}

func matchMapRealInteger(a, b map[float64]int64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[float64]int64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapRealBooleanIfNonZero(a, b map[float64]bool) bool {
	if b == nil {
		return true
	}
	return matchMapRealBoolean(a, b)
}

func matchMapRealBoolean(a, b map[float64]bool) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[float64]bool{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapRealRealIfNonZero(a, b map[float64]float64) bool {
	if b == nil {
		return true
	}
	return matchMapRealReal(a, b)
}

func matchMapRealReal(a, b map[float64]float64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[float64]float64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchUuidIfNonZero(a, b string) bool {
	var z string
	if b == z {
		return true
	}
	return matchUuid(a, b)
}

func matchUuid(a, b string) bool {
	return a == b
}

func matchUuidOptionalIfNonZero(a, b *string) bool {
	if b == nil {
		return true
	}
	return matchUuidOptional(a, b)
}

func matchUuidOptional(a, b *string) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func matchUuidMultiplesIfNonZero(a, b []string) bool {
	if b == nil {
		return true
	}
	return matchUuidMultiples(a, b)
}

func matchUuidMultiples(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := make([]string, len(b))
	copy(bCopy, b)
	for _, elA := range a {
		for i := len(bCopy) - 1; i >= 0; i-- {
			elB := bCopy[i]
			if elA == elB {
				bCopy = append(bCopy[:i], bCopy[i+1:]...)
			}
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapUuidBooleanIfNonZero(a, b map[string]bool) bool {
	if b == nil {
		return true
	}
	return matchMapUuidBoolean(a, b)
}

func matchMapUuidBoolean(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]bool{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapUuidRealIfNonZero(a, b map[string]float64) bool {
	if b == nil {
		return true
	}
	return matchMapUuidReal(a, b)
}

func matchMapUuidReal(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]float64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapUuidUuidIfNonZero(a, b map[string]string) bool {
	if b == nil {
		return true
	}
	return matchMapUuidUuid(a, b)
}

func matchMapUuidUuid(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapUuidStringIfNonZero(a, b map[string]string) bool {
	if b == nil {
		return true
	}
	return matchMapUuidString(a, b)
}

func matchMapUuidString(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapUuidIntegerIfNonZero(a, b map[string]int64) bool {
	if b == nil {
		return true
	}
	return matchMapUuidInteger(a, b)
}

func matchMapUuidInteger(a, b map[string]int64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]int64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchStringIfNonZero(a, b string) bool {
	var z string
	if b == z {
		return true
	}
	return matchString(a, b)
}

func matchString(a, b string) bool {
	return a == b
}

func matchStringOptionalIfNonZero(a, b *string) bool {
	if b == nil {
		return true
	}
	return matchStringOptional(a, b)
}

func matchStringOptional(a, b *string) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func matchStringMultiplesIfNonZero(a, b []string) bool {
	if b == nil {
		return true
	}
	return matchStringMultiples(a, b)
}

func matchStringMultiples(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := make([]string, len(b))
	copy(bCopy, b)
	for _, elA := range a {
		for i := len(bCopy) - 1; i >= 0; i-- {
			elB := bCopy[i]
			if elA == elB {
				bCopy = append(bCopy[:i], bCopy[i+1:]...)
			}
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapStringStringIfNonZero(a, b map[string]string) bool {
	if b == nil {
		return true
	}
	return matchMapStringString(a, b)
}

func matchMapStringString(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapStringIntegerIfNonZero(a, b map[string]int64) bool {
	if b == nil {
		return true
	}
	return matchMapStringInteger(a, b)
}

func matchMapStringInteger(a, b map[string]int64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]int64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapStringBooleanIfNonZero(a, b map[string]bool) bool {
	if b == nil {
		return true
	}
	return matchMapStringBoolean(a, b)
}

func matchMapStringBoolean(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]bool{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapStringRealIfNonZero(a, b map[string]float64) bool {
	if b == nil {
		return true
	}
	return matchMapStringReal(a, b)
}

func matchMapStringReal(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]float64{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}

func matchMapStringUuidIfNonZero(a, b map[string]string) bool {
	if b == nil {
		return true
	}
	return matchMapStringUuid(a, b)
}

func matchMapStringUuid(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	bCopy := map[string]string{}
	for k, v := range b {
		bCopy[k] = v
	}
	for aK, aV := range a {
		if bV, ok := bCopy[aK]; !ok || aV != bV {
			return false
		} else {
			delete(bCopy, aK)
		}
	}
	if len(bCopy) == 0 {
		return true
	}
	return false
}
