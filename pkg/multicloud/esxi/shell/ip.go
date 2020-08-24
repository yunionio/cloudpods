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
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type IPOption struct {
		Host bool
	}
	shellutils.R(&IPOption{}, "ip-all", "List all ip", func(cli *esxi.SESXiClient, args *IPOption) error {
		hostIps, _, err := cli.AllHostIP()
		if err != nil {
			return err
		}
		for name, ip := range hostIps {
			fmt.Printf("name: %s, ip: %s\n", name, ip)
		}
		// for i := range hosts {
		// 	fmt.Printf("host %s: \n", hosts[i].Name)
		// 	vmips, err := cli.VMIP(hosts[i])
		// 	if err != nil {
		// 		return err
		// 	}
		// 	for name, ip := range vmips {
		// 		fmt.Printf("\tname: %s, ip: %s\n", name, ip)
		// 	}
		// }
		vmips, err := cli.VMIP2()
		if err != nil {
			return err
		}
		for name, ip := range vmips {
			fmt.Printf("\tname: %s, ip: %s\n", name, ip)
		}
		return nil
	})
}
