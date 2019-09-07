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
	ResResults modulebase.ResourceManager
)

func init() {
	ResResults = NewMeterManager("res_result", "res_results",
		[]string{"res_id", "res_name", "cpu", "mem", "sys_disk", "data_disk", "ips", "res_type", "band_width", "os_distribution", "os_version", "platform", "region_id",
			"project_name", "user_name", "start_time", "end_time", "time_length", "cpu_amount", "mem_amount", "disk_amount", "baremetal_amount", "gpu_amount", "res_fee"},
		[]string{},
	)
	register(&ResResults)
}
