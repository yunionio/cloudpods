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
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type MachineTypeListOptions struct {
		ZONE       string
		MaxResults int
		PageToken  string
	}
	shellutils.R(&MachineTypeListOptions{}, "machine-type-list", "List machinetypes", func(cli *google.SRegion, args *MachineTypeListOptions) error {
		machinetypes, err := cli.GetMachineTypes(args.ZONE, args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(machinetypes, 0, 0, 0, nil)
		return nil
	})

	type MachineTypeShowOptions struct {
		ID string
	}
	shellutils.R(&MachineTypeShowOptions{}, "machine-type-show", "Show machinetype", func(cli *google.SRegion, args *MachineTypeShowOptions) error {
		machinetype, err := cli.GetMachineType(args.ID)
		if err != nil {
			return err
		}
		printObject(machinetype)
		return nil
	})

}
