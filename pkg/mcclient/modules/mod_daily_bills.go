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
	DailyBills modulebase.ResourceManager
)

func init() {
	DailyBills = NewMeterManager("daily_bill", "daily_bills",
		[]string{"account", "account_id", "charge_type", "region", "region_id", "domain", "domain_id", "project",
			"tenant_id", "brand", "resource_id", "resource_type", "resource_name", "rate", "reserved", "spec",
			"usage_type", "associate_id", "day", "price_unit", "currency", "cloudprovider_id", "cloudprovider_name",
			"amount", "gross_amount", "usage"},
		[]string{},
	)
	register(&DailyBills)
}
