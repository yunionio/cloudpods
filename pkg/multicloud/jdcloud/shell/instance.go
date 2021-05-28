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
	"yunion.io/x/onecloud/pkg/multicloud/jdcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		ListOptions
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *jdcloud.SRegion, args *InstanceListOptions) error {
		instances, total, e := cli.GetInstances("", nil, args.Offset+1, args.Limit)
		if e != nil {
			return e
		}
		printList(instances, total, args.Offset, args.Limit, []string{})
		return nil
	})
	type InstanceShowOptions struct {
		ID string
	}
	shellutils.R(&InstanceShowOptions{}, "instance-show", "Show intances", func(cli *jdcloud.SRegion, args *InstanceShowOptions) error {
		instance, e := cli.GetInstanceById(args.ID)
		if e != nil {
			return e
		}
		printObject(instance)
		return nil
	})
	shellutils.R(&InstanceShowOptions{}, "instance-nic-list", "List intance nics", func(cli *jdcloud.SRegion, args *InstanceShowOptions) error {
		instance, e := cli.GetInstanceById(args.ID)
		if e != nil {
			return e
		}
		cli.FillHost(instance)
		nics, err := instance.GetINics()
		if err != nil {
			return err
		}
		printList(nics, 0, 0, 0, []string{})
		return nil
	})
	shellutils.R(&InstanceShowOptions{}, "instance-disk-list", "List intance disks", func(cli *jdcloud.SRegion, args *InstanceShowOptions) error {
		instance, e := cli.GetInstanceById(args.ID)
		if e != nil {
			return e
		}
		cli.FillHost(instance)
		disks, err := instance.GetIDisks()
		if err != nil {
			return err
		}
		printList(disks, 0, 0, 0, []string{})
		return nil
	})
}
