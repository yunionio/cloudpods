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

package algorithmprovider

import (
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates"
	predicatebm "yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates/baremetal"
	"yunion.io/x/onecloud/pkg/scheduler/factory"
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
		factory.RegisterFitPredicate("f-BaremetalNetFilter", &predicates.NetworkPredicate{}),
		factory.RegisterFitPredicate("g-BaremetalResourceTypeFilter", &predicates.ResourceTypePredicate{}),
		factory.RegisterFitPredicate("h-DiskschedtagFilter", &predicates.DiskSchedtagPredicate{}),
		factory.RegisterFitPredicate("i-NetschedtagFilter", &predicates.NetworkSchedtagPredicate{}),
		factory.RegisterFitPredicate("k-NetBondingFilter", &predicatebm.NetBondingPredicate{}),
	)
}
