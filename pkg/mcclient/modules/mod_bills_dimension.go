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

package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

type SBillsDimensionManager struct {
	*modulebase.ResourceManager
}

var (
	BillingDimensionManager         *SBillsDimensionManager
	BillingDimensionAnalysisManager *SBillsDimensionManager
	BillingDimensionJointManager    *SBillsDimensionManager
)

func NewBillingDimensionManager() *SBillsDimensionManager {
	man := NewMeterManager("billsdimension", "billsdimensions",
		[]string{"id", "name", "dimension_type", "dimension_items"},
		[]string{})
	return &SBillsDimensionManager{
		ResourceManager: &man,
	}
}

func NewBillingDimensionAnalysisManager() *SBillsDimensionManager {
	man := NewMeterManager("billsdimensionsanalysis", "billsdimensionsanalysis",
		[]string{"id", "name", "usage_type", "resource_type", "brand", "description"},
		[]string{})
	return &SBillsDimensionManager{
		ResourceManager: &man,
	}
}

func NewBillingDimensionJointManager() *SBillsDimensionManager {
	man := NewMeterManager("dimensionjoint", "dimensionjoints",
		[]string{"id", "name", "data"},
		[]string{})
	return &SBillsDimensionManager{
		ResourceManager: &man,
	}
}

func init() {
	BillingDimensionManager = NewBillingDimensionManager()
	BillingDimensionAnalysisManager = NewBillingDimensionAnalysisManager()
	BillingDimensionJointManager = NewBillingDimensionJointManager()
	register(BillingDimensionManager)
	register(BillingDimensionAnalysisManager)
	register(BillingDimensionJointManager)
}
