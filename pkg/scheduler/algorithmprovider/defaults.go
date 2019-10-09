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
	predicateguest "yunion.io/x/onecloud/pkg/scheduler/algorithm/predicates/guest"
	priorityguest "yunion.io/x/onecloud/pkg/scheduler/algorithm/priorities/guest"
	"yunion.io/x/onecloud/pkg/scheduler/factory"
)

func init() {
	factory.RegisterAlgorithmProvider(factory.DefaultProvider, defaultPredicates(), defaultPriorities())
}

func defaultPredicates() sets.String {
	return sets.NewString(
		factory.RegisterFitPredicate("a-GuestHostStatusFilter", &predicateguest.StatusPredicate{}),
		factory.RegisterFitPredicate("b-GuestHypervisorFilter", &predicateguest.HypervisorPredicate{}),
		factory.RegisterFitPredicate("c-GuestAggregateFilter", &predicates.AggregatePredicate{}),
		factory.RegisterFitPredicate("d-GuestMigrateFilter", &predicateguest.MigratePredicate{}),
		factory.RegisterFitPredicate("e-GuestDomainFilter", &predicates.DomainPredicate{}),
		factory.RegisterFitPredicate("e-GuestImageFilter", &predicateguest.ImagePredicate{}),
		//factory.RegisterFitPredicate("f-GuestGroupFilter", &predicateguest.GroupPredicate{}),
		factory.RegisterFitPredicate("g-GuestCPUFilter", &predicateguest.CPUPredicate{}),
		factory.RegisterFitPredicate("h-GuestMemoryFilter", &predicateguest.MemoryPredicate{}),
		factory.RegisterFitPredicate("i-GuestStorageFilter", &predicateguest.StoragePredicate{}),
		factory.RegisterFitPredicate("j-GuestNetworkFilter", &predicates.NetworkPredicate{}),
		factory.RegisterFitPredicate("k-GuestIsolatedDeviceFilter", &predicateguest.IsolatedDevicePredicate{}),
		factory.RegisterFitPredicate("l-GuestResourceTypeFilter", &predicates.ResourceTypePredicate{}),
		factory.RegisterFitPredicate("m-GuestDiskschedtagFilter", &predicates.DiskSchedtagPredicate{}),
		factory.RegisterFitPredicate("n-ServerSkuFilter", &predicates.InstanceTypePredicate{}),
		factory.RegisterFitPredicate("o-GuestNetschedtagFilter", &predicates.NetworkSchedtagPredicate{}),
		factory.RegisterFitPredicate("p-GuestDispersionFilter", &predicates.InstanceGroupPredicate{}),
	)
}

func defaultPriorities() sets.String {
	return sets.NewString(
		factory.RegisterPriority("guest-avoid-same-host", &priorityguest.AvoidSameHostPriority{}, 1),
		factory.RegisterPriority("guest-lowload", &priorityguest.LowLoadPriority{}, 1),
		factory.RegisterPriority("guest-creating", &priorityguest.CreatingPriority{}, 1),
		factory.RegisterPriority("guest-capacity", &priorityguest.CapacityPriority{}, 1),
	)
}
