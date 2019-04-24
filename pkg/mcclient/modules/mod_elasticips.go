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
	Elasticips ResourceManager
)

func init() {
	Elasticips = NewComputeManager("eip", "eips",
		[]string{"ID", "Name", "IP_Addr", "Status",
			"Associate_Type", "Associate_ID",
			"Associate_Name",
			"Bandwidth", "Charge_Type",
		},
		[]string{})

	registerCompute(&Elasticips)
}
