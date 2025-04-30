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

package yunionconf

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	api "yunion.io/x/onecloud/pkg/apis/yunionconf"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
	options "yunion.io/x/onecloud/pkg/mcclient/options/yunionconf"
)

func init() {
	cmd := shell.NewResourceCmd(&yunionconf.ScopedPolicies)
	cmd.List(new(options.ScopedPolicyListOptions))
	cmd.Show(new(options.ScopedPolicyShowOptions))
	cmd.Create(new(options.ScopedPolicyCreateOptions))
	cmd.Update(new(options.ScopedPolicyUpdateOptions))
	R(&options.ScopedPolicyUpdateOptions{}, "scopedpolicy-edit", "edit scoped policy", func(s *mcclient.ClientSession, args *options.ScopedPolicyUpdateOptions) error {
		policyJson, err := yunionconf.ScopedPolicies.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		valueJson, err := policyJson.Get("policies")
		if err != nil {
			return err
		}
		valueYaml := valueJson.YAMLString()
		valueYaml, err = shellutils.Edit(valueYaml)
		if err != nil {
			return err
		}
		valueJson, err = jsonutils.ParseYAML(valueYaml)
		if err != nil {
			return err
		}
		input := api.ScopedPolicyUpdateInput{
			Policies: valueJson,
		}
		policyJson, err = yunionconf.ScopedPolicies.Update(s, args.ID, jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(policyJson)
		return nil
	})
	cmd.Delete(new(options.ScopedPolicyIDOptions))
	cmd.Perform("bind", new(options.ScopedPolicyBindOptions))
}
