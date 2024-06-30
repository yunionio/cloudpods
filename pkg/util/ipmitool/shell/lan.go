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

package shell

import (
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
)

func init() {
	type LanOptions struct {
		CHANNEL uint8 `help:"lan channel"`
	}
	shellutils.R(&LanOptions{}, "set-lan-dhcp", "Set lan channel DHCP", func(client ipmitool.IPMIExecutor, args *LanOptions) error {
		return ipmitool.SetLanDHCP(client, args.CHANNEL)
	})

	shellutils.R(&LanOptions{}, "get-lan-config", "Get lan channel config", func(cli ipmitool.IPMIExecutor, args *LanOptions) error {
		config, err := ipmitool.GetLanConfig(cli, args.CHANNEL)
		if err != nil {
			return err
		}
		printutils.PrintInterfaceObject(config)
		return nil
	})

	type SetLanStaticIpOptions struct {
		LanOptions
		IP      string
		MASK    string
		GATEWAY string
	}
	shellutils.R(&SetLanStaticIpOptions{}, "set-lan-static", "Set lan static network", func(cli ipmitool.IPMIExecutor, args *SetLanStaticIpOptions) error {
		return ipmitool.SetLanStatic(cli, args.CHANNEL, args.IP, args.MASK, args.GATEWAY)
	})
}
