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
	"yunion.io/x/onecloud/pkg/multicloud/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type PortListOptions struct {
		MacAddr  string
		DeviceId string
	}
	shellutils.R(&PortListOptions{}, "port-list", "List ports", func(cli *openstack.SRegion, args *PortListOptions) error {
		ports, err := cli.GetPorts(args.MacAddr, args.DeviceId)
		if err != nil {
			return err
		}
		printList(ports, 0, 0, 0, nil)
		return nil
	})

	type PortIdOptions struct {
		ID string
	}

	shellutils.R(&PortIdOptions{}, "port-show", "Show port", func(cli *openstack.SRegion, args *PortIdOptions) error {
		port, err := cli.GetPort(args.ID)
		if err != nil {
			return err
		}
		printObject(port)
		return nil
	})

}
