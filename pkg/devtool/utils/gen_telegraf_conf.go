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

package utils

import (
	"fmt"

	"yunion.io/x/pkg/util/osprofile"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/telegraf"
)

func GenerateTelegrafConf(
	serverDetails *api.ServerDetails, influxdbUrl, osType, hypervisor string,
) (string, error) {
	profile := "guest_linux"
	logfile := ""
	if osType == osprofile.OS_TYPE_WINDOWS {
		profile = "guest_windows"
		logfile = telegraf.RegionWindowsLogfile()
	}
	if hypervisor == api.HYPERVISOR_BAREMETAL {
		profile = "baremetal_linux"
	}
	args := GetLocalArgs(serverDetails, influxdbUrl)
	tags, ok := args["telegraf_global_tags"].([]map[string]string)
	if !ok {
		return "", fmt.Errorf("unexpected telegraf tags type %T", args["telegraf_global_tags"])
	}
	return telegraf.Render(profile, tags, influxdbUrl, logfile)
}
