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
	"yunion.io/x/onecloud/pkg/multicloud/apsara"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoadbalancerListenerRuleListOptions struct {
		ID   string `help:"ID of loadbalaner"`
		PORT int    `help:"Port of listener port"`
	}
	shellutils.R(&LoadbalancerListenerRuleListOptions{}, "lb-listener-rule-list", "List LoadbalancerListenerRules", func(cli *apsara.SRegion, args *LoadbalancerListenerRuleListOptions) error {
		rules, err := cli.GetLoadbalancerListenerRules(args.ID, args.PORT)
		if err != nil {
			return err
		}
		printList(rules, len(rules), 0, 0, []string{})
		return nil
	})
}
