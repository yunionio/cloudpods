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
	"yunion.io/x/onecloud/pkg/multicloud/incloudsphere"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		HOST_ID string
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "list instances", func(cli *incloudsphere.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstances(args.HOST_ID)
		if err != nil {
			return err
		}
		printList(instances, 0, 0, 0, []string{})
		return nil
	})

	type InstanceIdOptions struct {
		ID string
	}

	shellutils.R(&InstanceIdOptions{}, "instance-show", "show instance", func(cli *incloudsphere.SRegion, args *InstanceIdOptions) error {
		ret, err := cli.GetInstance(args.ID)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
