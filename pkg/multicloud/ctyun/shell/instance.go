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
	"yunion.io/x/onecloud/pkg/multicloud/ctyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		Id string `help:"ID of instance to show"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *ctyun.SRegion, args *InstanceListOptions) error {
		instances, e := cli.GetInstances(args.Id)
		if e != nil {
			return e
		}
		printList(instances, 0, 0, 0, []string{})
		return nil
	})

	type InstanceCreateOptions struct {
		ZoneId     string `help:"zone ID of instance"`
		NAME       string `help:"name of instance"`
		ADMINPASS  string `help:"admin password of instance"`
		ImageId    string `help:"image Id of instance"`
		OsType     string `help:"Os type of image"`
		VolumeType string `help:"volume type of instance"`
		VolumeSize int    `help:"volume size(GB) of instance"`
		Flavor     string `help:"Flavor of instance"`
		VpcId      string `help:"Vpc of instance"`
		SubnetId   string `help:"subnet Id of instance"`
		SecGroupId string `help:"security group Id of instance"`
	}

	shellutils.R(&InstanceCreateOptions{}, "instance-create", "Create intance", func(cli *ctyun.SRegion, args *InstanceCreateOptions) error {
		_, e := cli.CreateInstance(args.ZoneId, args.NAME, args.ImageId, args.OsType, args.Flavor, args.VpcId, args.SubnetId, args.SecGroupId, args.ADMINPASS, args.VolumeType, args.VolumeSize, nil)
		if e != nil {
			return e
		}

		return nil
	})
}
