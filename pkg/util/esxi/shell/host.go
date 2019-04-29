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
	"yunion.io/x/onecloud/pkg/util/esxi"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type HostListOptions struct {
		DATACENTER string `help:"List hosts in datacenter"`
	}
	shellutils.R(&HostListOptions{}, "host-list", "List hosts in datacenter", func(cli *esxi.SESXiClient, args *HostListOptions) error {
		dc, err := cli.FindDatacenterByMoId(args.DATACENTER)
		if err != nil {
			return err
		}
		hosts, err := dc.GetIHosts()
		if err != nil {
			return err
		}
		printList(hosts, nil)
		return nil
	})

	type HostShowOptions struct {
		IP string `help:"Host IP"`
	}
	shellutils.R(&HostShowOptions{}, "host-show", "Show details of a host by IP", func(cli *esxi.SESXiClient, args *HostShowOptions) error {
		host, err := cli.FindHostByIp(args.IP)
		if err != nil {
			return err
		}
		printObject(host)
		return nil
	})

	shellutils.R(&HostShowOptions{}, "host-storages", "Show all storages of a given host", func(cli *esxi.SESXiClient, args *HostShowOptions) error {
		host, err := cli.FindHostByIp(args.IP)
		if err != nil {
			return err
		}
		storages, err := host.GetIStorages()
		if err != nil {
			return err
		}
		printList(storages, nil)
		return nil
	})

	shellutils.R(&HostShowOptions{}, "host-nics", "Show all nics of a given host", func(cli *esxi.SESXiClient, args *HostShowOptions) error {
		host, err := cli.FindHostByIp(args.IP)
		if err != nil {
			return err
		}
		nics, err := host.GetIHostNics()
		if err != nil {
			return err
		}
		printList(nics, nil)
		return nil
	})
}
