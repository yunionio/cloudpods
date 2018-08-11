package algorithmprovider

import (
	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	predicatebm "yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates/baremetal"
	"yunion.io/x/onecloud/pkg/scheduler/factory"
	"yunion.io/x/pkg/util/sets"
)

func init() {
	// Register BaremetalProvider algorithm provider
	factory.RegisterAlgorithmProvider(factory.BaremetalProvider, baremetalPredicates(), nil)
}

func baremetalPredicates() sets.String {
	return sets.NewString(
		factory.RegisterFitPredicate("a-BaremetalStatusFilter", &predicatebm.StatusPredicate{}),
		factory.RegisterFitPredicate("b-BaremetalAggregateFilter", &predicates.AggregatePredicate{}),
		factory.RegisterFitPredicate("c-BaremetalCPUFilter", &predicatebm.CPUPredicate{}),
		factory.RegisterFitPredicate("d-BaremetalMemoryFilter", &predicatebm.MemoryPredicate{}),
		factory.RegisterFitPredicate("e-BaremetalStorageFilter", &predicatebm.StoragePredicate{}),
		factory.RegisterFitPredicate("f-BaremetalNetFilter", &predicatebm.NetworkPredicate{}),
	)
}
