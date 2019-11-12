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
	type SecurityGroupListOptions struct {
		Name       string
		SecgroupId string
		InstanceId string
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List secgroups", func(cli *zstack.SRegion, args *SecurityGroupListOptions) error {
		secgroups, err := cli.GetSecurityGroups(args.SecgroupId, args.InstanceId, args.Name)
		if err != nil {
			return err
		}
		printList(secgroups, len(secgroups), 0, 0, []string{})
		return nil
	})
}
