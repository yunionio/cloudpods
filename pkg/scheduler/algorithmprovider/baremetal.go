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
		factory.RegisterFitPredicate("b-BaremetalschedtagFilter", predicates.NewHostSchedtagPredicate()),
		factory.RegisterFitPredicate("c-BaremetalCPUFilter", &predicatebm.CPUPredicate{}),
		factory.RegisterFitPredicate("d-BaremetalMemoryFilter", &predicatebm.MemoryPredicate{}),
		factory.RegisterFitPredicate("e-BaremetalStorageFilter", &predicatebm.StoragePredicate{}),
		factory.RegisterFitPredicate("f-BaremetalNetFilter", predicates.NewNetworkPredicateWithNicCounter()),
		factory.RegisterFitPredicate("g-BaremetalResourceTypeFilter", &predicates.ResourceTypePredicate{}),
		factory.RegisterFitPredicate("h-DiskschedtagFilter", &predicates.DiskSchedtagPredicate{}),
		factory.RegisterFitPredicate("i-NetschedtagFilter", predicates.NewNetworkSchedtagPredicate()),
		factory.RegisterFitPredicate("k-NetBondingFilter", &predicatebm.NetBondingPredicate{}),
		factory.RegisterFitPredicate("l-CdromFilter", &predicatebm.CdromBootPredicate{}),
		factory.RegisterFitPredicate("m-IsolatedDevicesFilter", &predicates.IsolatedDevicePredicate{}),
		factory.RegisterFitPredicate("n-CloudproviderschedtagFilter", predicates.NewCloudproviderSchedtagPredicate()),
		factory.RegisterFitPredicate("o-CloudregionschedtagFilter", predicates.NewCloudregionSchedtagPredicate()),
		factory.RegisterFitPredicate("p-ZoneschedtagFilter", predicates.NewZoneSchedtagPredicate()),
		factory.RegisterFitPredicate("q-UEFIFilter", &predicatebm.UEFIImagePredicate{}),
	)
}
