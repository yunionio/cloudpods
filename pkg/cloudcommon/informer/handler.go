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

package informer

import (
	"sync"

	"yunion.io/x/pkg/util/sets"
)

var (
	globalWatchResources = new(sync.Map)
)

func AddWatchedResources(resources ...string) {
	for _, res := range resources {
		globalWatchResources.Store(res, true)
	}
}

func DeleteWatchedResources(resources ...string) {
	for _, res := range resources {
		globalWatchResources.Delete(res)
	}
}

func GetWatchResources() sets.String {
	ret := sets.NewString()
	globalWatchResources.Range(func(key, val interface{}) bool {
		ret.Insert(key.(string))
		return true
	})
	return ret
}
