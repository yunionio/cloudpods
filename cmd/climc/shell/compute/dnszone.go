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

package compute

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.DnsZones).WithKeyword("dns-zone")
	cmd.List(&compute.SDnsZoneListOptions{})
	cmd.Show(&compute.SDnsZoneIdOptions{})
	cmd.ClassShow(&compute.DnsZoneCapabilitiesOptions{})
	cmd.Delete(&compute.SDnsZoneIdOptions{})
	cmd.Create(&compute.DnsZoneCreateOptions{})
	cmd.Perform("public", &options.BasePublicOptions{})
	cmd.Perform("private", &options.BaseIdOptions{})
	cmd.Perform("syncstatus", &compute.SDnsZoneIdOptions{})
	cmd.Perform("purge", &compute.SDnsZoneIdOptions{})
	cmd.Perform("add-vpcs", &compute.DnsZoneAddVpcsOptions{})
	cmd.Perform("remove-vpcs", &compute.DnsZoneRemoveVpcsOptions{})
	cmd.GetWithCustomShow("exports", func(result jsonutils.JSONObject) {
		rr := make(map[string]string)
		err := result.Unmarshal(&rr)
		if err != nil {
			log.Errorf("error: %v", err)
			return
		}
		for _, v := range rr {
			fmt.Printf("%s\n", v)
		}
	}, &compute.SDnsZoneIdOptions{})
}
