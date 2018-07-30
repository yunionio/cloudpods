package factory

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
	"github.com/yunionio/pkg/util/sets"
)

type AlgorithmProviderConfig struct {
	FitPredicateKeys sets.String
	PriorityKeys     sets.String
}

type FitPredicateFactory func() core.FitPredicate

type PriorityFunctionFactory func() (core.PriorityPreFunction, core.PriorityMapFunction, core.PriorityReduceFunction)

type PriorityConfigFactory struct {
	MapReduceFunction PriorityFunctionFactory
	Weight            int
}

var (
	schedulerFactoryMutex sync.Mutex

	// maps that hold registered algorithm types
	fitPredicateMap      = make(map[string]FitPredicateFactory)
	priorityConfigMap    = make(map[string]PriorityConfigFactory)
	algorithmProviderMap = make(map[string]AlgorithmProviderConfig)

	validName = regexp.MustCompile("^[a-zA-Z0-9]([-a-zA-Z0-9]*[a-zA-Z0-9])$")
)

const (
	DefaultProvider   = "DefaultProvider"
	BaremetalProvider = "BaremetalProvider"
)

// RegisterAlgorithmProvider registers a new algorithm provider with the
// algorithm registry. This shoud be called from the init function in a
// provider plugin.
func RegisterAlgorithmProvider(name string, predicatesKeys, priorityKeys sets.String) string {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()
	validateAlgorithmNameOrDie(name)
	algorithmProviderMap[name] = AlgorithmProviderConfig{
		FitPredicateKeys: predicatesKeys,
		PriorityKeys:     priorityKeys,
	}
	return name
}

func validateAlgorithmNameOrDie(name string) {
	if !validName.MatchString(name) {
		log.Fatalf("Algorithm name %v does not match the name validation regexp \"%v\".", name, validName)
	}

}

// RegisterFitPredicate registers a fit predicate with the algorithm
// registry. Returns the name with which the predicates was registerd.
func RegisterFitPredicate(name string, predicate core.FitPredicate) string {
	return RegisterFitPredicateFactory(name, func() core.FitPredicate { return predicate })
}

// RegisterFitPredicateFactory registers a fit predicate factory with the
// algorithm registry. Returns the name with which the predicate was registered.
func RegisterFitPredicateFactory(name string, predicateFactory FitPredicateFactory) string {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()
	validateAlgorithmNameOrDie(name)
	fitPredicateMap[name] = predicateFactory
	return name
}

func getFitPredites(names sets.String) (map[string]core.FitPredicate, error) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	predicates := map[string]core.FitPredicate{}
	for _, name := range names.List() {
		factory, ok := fitPredicateMap[name]
		if !ok {
			return nil, fmt.Errorf("Invalid predicate name %q specified - no corresponding predicate found", name)
		}
		predicates[name] = factory()
	}
	return predicates, nil
}

// RegisterPriority registers a priority with the algorithm registry.
func RegisterPriority(name string, priority core.Priority, weight int) string {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()
	validateAlgorithmNameOrDie(name)
	priorityConfigMap[name] = PriorityConfigFactory{
		MapReduceFunction: func() (core.PriorityPreFunction, core.PriorityMapFunction, core.PriorityReduceFunction) {
			p := priority.Clone()
			return p.PreExecute, p.Map, p.Reduce
		},
		Weight: weight,
	}
	return name
}

func getPriorityConfigs(names sets.String) ([]core.PriorityConfig, error) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	configs := []core.PriorityConfig{}
	for _, name := range names.List() {
		factory, ok := priorityConfigMap[name]
		if !ok {
			return nil, fmt.Errorf("Invalid priority name %q specified - no corresponding priority found", name)
		}
		preFunc, mapFunc, reduceFunc := factory.MapReduceFunction()
		configs = append(configs, core.PriorityConfig{
			Pre:    preFunc,
			Map:    mapFunc,
			Reduce: reduceFunc,
			Weight: factory.Weight,
		})
	}
	return configs, nil
}
