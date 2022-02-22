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
	type EipListOptions struct {
	}

	shellutils.R(&EipListOptions{}, "eip-list", "list eips", func(cli *bingocloud.SRegion, args *EipListOptions) error {
		eips, total, err := cli.GetEips()
		if err != nil {
			return err
		}
		printList(eips, total, 0, 0, []string{})
		return nil
	})

	type EipIdOptions struct {
		ID string
	}

	shellutils.R(&EipIdOptions{}, "eip-show", "eip instance", func(cli *bingocloud.SRegion, args *EipIdOptions) error {
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
