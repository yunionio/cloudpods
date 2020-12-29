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

var (
	Budgets modulebase.ResourceManager
)

func init() {
	Budgets = NewMeterManager("budget", "budgets",
		[]string{"period_type", "start_time", "end_time", "brand", "cloudaccount_id", "cloudaccount",
			"cloudprovider_id", "cloudprovider_name", "region_id", "region",
			"domain_id_filter", "domain_filter", "project_id_filter", "project_filter",
			"resource_type", "currency", "amount", "alerts", "availability_status", "budget_scope"},
		[]string{},
	)
	register(&Budgets)
}
