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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	compute_options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {

	cmd := shell.NewResourceCmd(&modules.NetworkIpMacs)
	cmd.List(&compute_options.NetworkIpMacListOptions{})
	cmd.Update(&compute_options.NetworkIpMacUpdateOptions{})
	cmd.Show(&compute_options.NetworkIpMacIdOptions{})
	cmd.Delete(&compute_options.NetworkIpMacIdOptions{})
	cmd.Create(&compute_options.NetworkIpMacCreateOptions{})
	type NetworkIpMacBatchCreateOptions struct {
		NETWORK string            `help:"network id" json:"network_id"`
		IpMac   map[string]string `help:"ip mac map" json:"ip_mac"`
	}
	R(&NetworkIpMacBatchCreateOptions{},
		"network-ip-mac-batch-create",
		"Network ip mac bind batch create",
		func(s *mcclient.ClientSession, args *NetworkIpMacBatchCreateOptions) error {
			params := jsonutils.Marshal(args)
			_, err := modules.NetworkIpMacs.PerformClassAction(s, "batch-create", params)
			return err
		},
	)
}
