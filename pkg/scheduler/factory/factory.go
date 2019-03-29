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

package factory

import (
	"fmt"

	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func GetAlgorithmProvider(name string) (*AlgorithmProviderConfig, error) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	provider, ok := algorithmProviderMap[name]
	if !ok {
		return nil, fmt.Errorf("AlgorithmProvider plugin %q has not been registered", name)
	}
	return &provider, nil
}

func GetPredicates(predicatesKeys sets.String) (map[string]core.FitPredicate, error) {
	return getFitPredites(predicatesKeys)
}

func GetPriorityConfigs(priorityKeys sets.String) ([]core.PriorityConfig, error) {
	return getPriorityConfigs(priorityKeys)
}
