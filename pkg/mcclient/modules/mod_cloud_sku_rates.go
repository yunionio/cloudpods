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

var (
	CloudSkuRates ResourceManager
)

func init() {
	CloudSkuRates = NewMeterManager("cloud_sku_rate", "cloud_sku_rates",
		[]string{"id", "data_id", "data_key", "hour_price", "month_price", "year_price"},
		[]string{},
	)
	register(&CloudSkuRates)
}
