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
	BillCloudChecks modulebase.ResourceManager
)

func init() {
	BillCloudChecks = NewMeterManager("bill_cloudcheck", "bill_cloudchecks",
		[]string{"provider", "account_id", "sum_month", "res_type", "res_id", "res_name", "external_id", "cloud_fee", "kvm_fee", "diff_fee", "diff_percent"},
		[]string{},
	)
	register(&BillCloudChecks)
}
