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
	"yunion.io/x/onecloud/pkg/multicloud/zstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceOfferingListOptions struct {
		OfferId  string
		Name     string
		Cpu      int
		MemoryMb int
	}
	shellutils.R(&InstanceOfferingListOptions{}, "instance-offering-list", "List instance offerings", func(cli *zstack.SRegion, args *InstanceOfferingListOptions) error {
		offerings, err := cli.GetInstanceOfferings(args.OfferId, args.Name, args.Cpu, args.MemoryMb)
		if err != nil {
			return err
		}
		printList(offerings, len(offerings), 0, 0, []string{})
		return nil
	})

	type InstanceOfferingCreateOptions struct {
		NAME      string
		CPU       int
		MEMORY_MB int
		TYPE      string `choices:"UserVm"`
	}

	shellutils.R(&InstanceOfferingCreateOptions{}, "instance-offering-create", "Create instance offerings", func(cli *zstack.SRegion, args *InstanceOfferingCreateOptions) error {
		offer, err := cli.CreateInstanceOffering(args.NAME, args.CPU, args.MEMORY_MB, args.TYPE)
		if err != nil {
			return err
		}
		printObject(offer)
		return nil
	})

	type InstanceOfferingIdOptions struct {
		ID string
	}

	shellutils.R(&InstanceOfferingIdOptions{}, "instance-offering-delete", "Delete instance offerings", func(cli *zstack.SRegion, args *InstanceOfferingIdOptions) error {
		return cli.DeleteOffering(args.ID)
	})

}
