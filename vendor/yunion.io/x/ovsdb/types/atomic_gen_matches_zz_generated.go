// DO NOT EDIT: automatically generated code

package types

func MatchStringIfNonZero(a, b string) bool {
	var z string
	if b == z {
		return true
	}
	return MatchString(a, b)
}

func MatchString(a, b string) bool {
	return a == b
}

func MatchStringOptionalIfNonZero(a, b *string) bool {
	if b == nil {
		return true
	}
	return MatchStringOptional(a, b)
}

func MatchStringOptional(a, b *string) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func MatchStringMultiplesIfNonZero(a, b []string) bool {
	if b == nil {
		return true
	}
	return MatchStringMultiples(a, b)
}

func MatchStringMultiples(a, b []string) bool {
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

func MatchMapStringStringIfNonZero(a, b map[string]string) bool {
	if b == nil {
		return true
	}
	return MatchMapStringString(a, b)
}

func MatchMapStringString(a, b map[string]string) bool {
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

func MatchMapStringIntegerIfNonZero(a, b map[string]int64) bool {
	if b == nil {
		return true
	}
	return MatchMapStringInteger(a, b)
}

func MatchMapStringInteger(a, b map[string]int64) bool {
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

func MatchMapStringRealIfNonZero(a, b map[string]float64) bool {
	if b == nil {
		return true
	}
	return MatchMapStringReal(a, b)
}

func MatchMapStringReal(a, b map[string]float64) bool {
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

func MatchMapStringBooleanIfNonZero(a, b map[string]bool) bool {
	if b == nil {
		return true
	}
	return MatchMapStringBoolean(a, b)
}

func MatchMapStringBoolean(a, b map[string]bool) bool {
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

func MatchMapStringUuidIfNonZero(a, b map[string]string) bool {
	if b == nil {
		return true
	}
	return MatchMapStringUuid(a, b)
}

func MatchMapStringUuid(a, b map[string]string) bool {
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

func MatchIntegerIfNonZero(a, b int64) bool {
	var z int64
	if b == z {
		return true
	}
	return MatchInteger(a, b)
}

func MatchInteger(a, b int64) bool {
	return a == b
}

func MatchIntegerOptionalIfNonZero(a, b *int64) bool {
	if b == nil {
		return true
	}
	return MatchIntegerOptional(a, b)
}

func MatchIntegerOptional(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func MatchIntegerMultiplesIfNonZero(a, b []int64) bool {
	if b == nil {
		return true
	}
	return MatchIntegerMultiples(a, b)
}

func MatchIntegerMultiples(a, b []int64) bool {
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

func MatchMapIntegerStringIfNonZero(a, b map[int64]string) bool {
	if b == nil {
		return true
	}
	return MatchMapIntegerString(a, b)
}

func MatchMapIntegerString(a, b map[int64]string) bool {
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

func MatchMapIntegerIntegerIfNonZero(a, b map[int64]int64) bool {
	if b == nil {
		return true
	}
	return MatchMapIntegerInteger(a, b)
}

func MatchMapIntegerInteger(a, b map[int64]int64) bool {
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

func MatchMapIntegerRealIfNonZero(a, b map[int64]float64) bool {
	if b == nil {
		return true
	}
	return MatchMapIntegerReal(a, b)
}

func MatchMapIntegerReal(a, b map[int64]float64) bool {
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

func MatchMapIntegerBooleanIfNonZero(a, b map[int64]bool) bool {
	if b == nil {
		return true
	}
	return MatchMapIntegerBoolean(a, b)
}

func MatchMapIntegerBoolean(a, b map[int64]bool) bool {
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

func MatchMapIntegerUuidIfNonZero(a, b map[int64]string) bool {
	if b == nil {
		return true
	}
	return MatchMapIntegerUuid(a, b)
}

func MatchMapIntegerUuid(a, b map[int64]string) bool {
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

func MatchRealIfNonZero(a, b float64) bool {
	var z float64
	if b == z {
		return true
	}
	return MatchReal(a, b)
}

func MatchReal(a, b float64) bool {
	return a == b
}

func MatchRealOptionalIfNonZero(a, b *float64) bool {
	if b == nil {
		return true
	}
	return MatchRealOptional(a, b)
}

func MatchRealOptional(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func MatchRealMultiplesIfNonZero(a, b []float64) bool {
	if b == nil {
		return true
	}
	return MatchRealMultiples(a, b)
}

func MatchRealMultiples(a, b []float64) bool {
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

func MatchMapRealStringIfNonZero(a, b map[float64]string) bool {
	if b == nil {
		return true
	}
	return MatchMapRealString(a, b)
}

func MatchMapRealString(a, b map[float64]string) bool {
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

func MatchMapRealIntegerIfNonZero(a, b map[float64]int64) bool {
	if b == nil {
		return true
	}
	return MatchMapRealInteger(a, b)
}

func MatchMapRealInteger(a, b map[float64]int64) bool {
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

func MatchMapRealRealIfNonZero(a, b map[float64]float64) bool {
	if b == nil {
		return true
	}
	return MatchMapRealReal(a, b)
}

func MatchMapRealReal(a, b map[float64]float64) bool {
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

func MatchMapRealBooleanIfNonZero(a, b map[float64]bool) bool {
	if b == nil {
		return true
	}
	return MatchMapRealBoolean(a, b)
}

func MatchMapRealBoolean(a, b map[float64]bool) bool {
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

func MatchMapRealUuidIfNonZero(a, b map[float64]string) bool {
	if b == nil {
		return true
	}
	return MatchMapRealUuid(a, b)
}

func MatchMapRealUuid(a, b map[float64]string) bool {
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

func MatchBooleanIfNonZero(a, b bool) bool {
	var z bool
	if b == z {
		return true
	}
	return MatchBoolean(a, b)
}

func MatchBoolean(a, b bool) bool {
	return a == b
}

func MatchBooleanOptionalIfNonZero(a, b *bool) bool {
	if b == nil {
		return true
	}
	return MatchBooleanOptional(a, b)
}

func MatchBooleanOptional(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func MatchBooleanMultiplesIfNonZero(a, b []bool) bool {
	if b == nil {
		return true
	}
	return MatchBooleanMultiples(a, b)
}

func MatchBooleanMultiples(a, b []bool) bool {
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

func MatchMapBooleanStringIfNonZero(a, b map[bool]string) bool {
	if b == nil {
		return true
	}
	return MatchMapBooleanString(a, b)
}

func MatchMapBooleanString(a, b map[bool]string) bool {
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

func MatchMapBooleanIntegerIfNonZero(a, b map[bool]int64) bool {
	if b == nil {
		return true
	}
	return MatchMapBooleanInteger(a, b)
}

func MatchMapBooleanInteger(a, b map[bool]int64) bool {
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

func MatchMapBooleanRealIfNonZero(a, b map[bool]float64) bool {
	if b == nil {
		return true
	}
	return MatchMapBooleanReal(a, b)
}

func MatchMapBooleanReal(a, b map[bool]float64) bool {
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

func MatchMapBooleanBooleanIfNonZero(a, b map[bool]bool) bool {
	if b == nil {
		return true
	}
	return MatchMapBooleanBoolean(a, b)
}

func MatchMapBooleanBoolean(a, b map[bool]bool) bool {
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

func MatchMapBooleanUuidIfNonZero(a, b map[bool]string) bool {
	if b == nil {
		return true
	}
	return MatchMapBooleanUuid(a, b)
}

func MatchMapBooleanUuid(a, b map[bool]string) bool {
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

func MatchUuidIfNonZero(a, b string) bool {
	var z string
	if b == z {
		return true
	}
	return MatchUuid(a, b)
}

func MatchUuid(a, b string) bool {
	return a == b
}

func MatchUuidOptionalIfNonZero(a, b *string) bool {
	if b == nil {
		return true
	}
	return MatchUuidOptional(a, b)
}

func MatchUuidOptional(a, b *string) bool {
	if a == nil && b == nil {
		return true
	} else if a != nil && b != nil {
		return *a == *b
	}
	return false
}

func MatchUuidMultiplesIfNonZero(a, b []string) bool {
	if b == nil {
		return true
	}
	return MatchUuidMultiples(a, b)
}

func MatchUuidMultiples(a, b []string) bool {
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

func MatchMapUuidStringIfNonZero(a, b map[string]string) bool {
	if b == nil {
		return true
	}
	return MatchMapUuidString(a, b)
}

func MatchMapUuidString(a, b map[string]string) bool {
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

func MatchMapUuidIntegerIfNonZero(a, b map[string]int64) bool {
	if b == nil {
		return true
	}
	return MatchMapUuidInteger(a, b)
}

func MatchMapUuidInteger(a, b map[string]int64) bool {
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

func MatchMapUuidRealIfNonZero(a, b map[string]float64) bool {
	if b == nil {
		return true
	}
	return MatchMapUuidReal(a, b)
}

func MatchMapUuidReal(a, b map[string]float64) bool {
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

func MatchMapUuidBooleanIfNonZero(a, b map[string]bool) bool {
	if b == nil {
		return true
	}
	return MatchMapUuidBoolean(a, b)
}

func MatchMapUuidBoolean(a, b map[string]bool) bool {
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

func MatchMapUuidUuidIfNonZero(a, b map[string]string) bool {
	if b == nil {
		return true
	}
	return MatchMapUuidUuid(a, b)
}

func MatchMapUuidUuid(a, b map[string]string) bool {
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
