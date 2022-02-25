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
	"yunion.io/x/onecloud/pkg/multicloud/bingocloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ZoneListOptions struct {
		// Id        string
		// MaxResult int
		// NextToken string
		Details bool `help:"show Details"`
	}

	shellutils.R(&ZoneListOptions{}, "zone-list", "list zones", func(cli *bingocloud.SRegion, args *ZoneListOptions) error {
		zones, err := cli.GetZones()
		if err != nil {
			return err
		}
		cols := []string{"zone_id", "local_name", "available_resource_creation", "available_disk_categories"}
		if args.Details {
			cols = []string{}
		}
		printList(zones, 0, 0, 0, cols)
		return nil
	})

	type InstanceIdOptions struct {
		ID string
	}

	shellutils.R(&InstanceIdOptions{}, "zone-show", "zone instance", func(cli *bingocloud.SRegion, args *InstanceIdOptions) error {
		/*
			vm, err := cli.GetInstance(args.ID)
			if err != nil {
				return err
			}
			printObject(vm)
		*/
		return nil
	})

}
