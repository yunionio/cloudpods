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
