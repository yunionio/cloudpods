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
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.DnsZones).WithKeyword("dns-zone")
	cmd.List(&options.SDnsZoneListOptions{})
	cmd.Show(&options.SDnsZoneIdOptions{})
	cmd.ClassShow(&options.DnsZoneCapabilitiesOptions{})
	cmd.Delete(&options.SDnsZoneIdOptions{})
	cmd.Create(&options.DnsZoneCreateOptions{})
	cmd.Perform("syncstatus", &options.SDnsZoneIdOptions{})
	cmd.Perform("sync-recordsets", &options.SDnsZoneIdOptions{})
	cmd.Perform("cache", &options.DnsZoneCacheOptions{})
	cmd.Perform("uncache", &options.DnsZoneUncacheOptions{})
	cmd.Perform("purge", &options.SDnsZoneIdOptions{})
	cmd.Perform("add-vpcs", &options.DnsZoneAddVpcsOptions{})
	cmd.Perform("remove-vpcs", &options.DnsZoneRemoveVpcsOptions{})
}
