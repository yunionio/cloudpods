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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ServiceListOptions struct {
	}
	shellutils.R(&ServiceListOptions{}, "service-list", "List providers", func(cli *azure.SRegion, args *ServiceListOptions) error {
		services, err := cli.GetClient().ListServices()
		if err != nil {
			return err
		}
		printList(services, len(services), 0, 0, []string{})
		return nil
	})

	type ServiceOptions struct {
		NAME string `help:"Name for service register"`
	}

	shellutils.R(&ServiceOptions{}, "service-register", "Register service", func(cli *azure.SRegion, args *ServiceOptions) error {
		return cli.GetClient().ServiceRegister(args.NAME)
	})

	shellutils.R(&ServiceOptions{}, "service-unregister", "Unregister service", func(cli *azure.SRegion, args *ServiceOptions) error {
		return cli.GetClient().ServiceUnRegister(args.NAME)
	})

	shellutils.R(&ServiceOptions{}, "service-show", "Show service detail", func(cli *azure.SRegion, args *ServiceOptions) error {
		service, err := cli.GetClient().GetSercice(args.NAME)
		if err != nil {
			return err
		}
		printObject(service)
		return nil
	})

}
