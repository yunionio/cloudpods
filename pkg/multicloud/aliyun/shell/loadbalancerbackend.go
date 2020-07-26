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
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerBackendListOptions struct {
		GROUPID string `help:"LoadbalancerBackendgroup ID"`
	}
	shellutils.R(&LoadbalancerBackendListOptions{}, "lb-backend-list", "List loadbalanceBackends", func(cli *aliyun.SRegion, args *LoadbalancerBackendListOptions) error {
		backends, err := cli.GetLoadbalancerBackends(args.GROUPID)
		if err != nil {
			return err
		}
		printList(backends, len(backends), 0, 0, []string{})
		return nil
	})

	shellutils.R(&LoadbalancerBackendListOptions{}, "lb-main-subordinate-backend-list", "List loadbalanceMainSubordinateBackends", func(cli *aliyun.SRegion, args *LoadbalancerBackendListOptions) error {
		backends, err := cli.GetLoadbalancerMainSubordinateBackends(args.GROUPID)
		if err != nil {
			return err
		}
		printList(backends, len(backends), 0, 0, []string{})
		return nil
	})
}
