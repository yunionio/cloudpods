package factory

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/pkg/util/sets"
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
