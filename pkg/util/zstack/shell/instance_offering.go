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
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
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
}
