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
	"yunion.io/x/onecloud/pkg/multicloud/proxmox"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type HostListOptions struct {
	}
	shellutils.R(&HostListOptions{}, "host-list", "list hosts", func(cli *proxmox.SRegion, args *HostListOptions) error {
		hosts, err := cli.GetHosts()
		if err != nil {
			return err
		}
		printList(hosts, 0, 0, 0, []string{})
		return nil
	})

	type HostIdOptions struct {
		ID string
	}

	shellutils.R(&HostIdOptions{}, "host-show", "show host", func(cli *proxmox.SRegion, args *HostIdOptions) error {
		host, err := cli.GetHost(args.ID)
		if err != nil {
			return err
		}
		printObject(host)
		printObject(host.Cpuinfo)
		return nil
	})

}
