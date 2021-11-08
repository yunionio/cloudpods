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

package vpcproxy

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/vpcproxy"
)

func init() {
	type VNCConnectOptions struct {
		ID        string `help:"ID of server to connect"`
		Baremetal bool   `help:"Connect to baremetal"`
	}
	R(&VNCConnectOptions{}, "vnc-connect", "Show the VNC console of given server", func(s *mcclient.ClientSession, args *VNCConnectOptions) error {
		params := jsonutils.NewDict()
		if args.Baremetal {
			params.Add(jsonutils.NewString("hosts"), "objtype")
		}
		result, err := vpcproxy.VNCProxy.DoConnect(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
