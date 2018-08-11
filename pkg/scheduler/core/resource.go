package core

import (
	"yunion.io/x/pkg/utils"
)

type value_t interface{}

type ResourceAlgorithm interface {
	Sum(values []value_t) value_t
	Sub(sum value_t, reserved value_t) value_t
}

type DefaultResAlgorithm struct{}

func (al *DefaultResAlgorithm) Sum(values []value_t) value_t {
	var ret int64 = 0
	for _, value := range values {
		if value != nil {
			ret += value.(int64)
		}
	}
	return ret
}

func (al *DefaultResAlgorithm) Sub(sum value_t, reserved value_t) value_t {
	if sum == nil {
		return nil
	}

	if reserved == nil {
		return sum
	}

	return sum.(int64) - reserved.(int64)
}

var (
	g_defaultResAlgorithm *DefaultResAlgorithm = &DefaultResAlgorithm{}
)

func GetResourceAlgorithm(res_name string) ResourceAlgorithm {
	switch res_name {
	//case "Groups":
	//return g_groupResAlgorithm
	case "FreeCPUCount", "FreeMemSize", "FreeLocalStorageSize", "Ports":
		return g_defaultResAlgorithm
	default:
		if utils.HasPrefix(res_name, "FreeStorageSize:") {
			return g_defaultResAlgorithm
		}

		return nil
	}
}

func ReservedSub(key string, value value_t, reserved value_t) value_t {
	al := GetResourceAlgorithm(key)
	if al != nil {
		return al.Sub(value, reserved)
	}
	return value
}

func XGetCalculator(c Candidater, key string, kind Kind) value_t {
	value := c.Get(key)

	switch kind {
	case KindFree:
		// TODO: reserved not impl by now
		return ReservedSub(key, value, nil)
	case KindRaw:
		return value
	case KindReserved:
		return nil
	}
	return nil
}
