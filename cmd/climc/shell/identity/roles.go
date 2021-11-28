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

package identity

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/identity"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.RolesV3)
	cmd.List(&identity.RoleListOptions{})
	cmd.Show(&identity.RoleDetailOptions{})
	cmd.Delete(&identity.RoleDetailOptions{})
	cmd.Create(&identity.RoleCreateOptions{})
	cmd.Perform("public", &options.BaseIdOptions{})
	cmd.Perform("private", &options.BaseIdOptions{})
	cmd.GetProperty(&identity.RoleGetPropertyTagValuePairOptions{})
	cmd.GetProperty(&identity.RoleGetPropertyTagValueTreeOptions{})
	cmd.GetProperty(&identity.RoleGetPropertyDomainTagValuePairOptions{})
	cmd.GetProperty(&identity.RoleGetPropertyDomainTagValueTreeOptions{})

	type RoleUpdateOptions struct {
		ID     string `help:"Role ID or Name"`
		Domain string `help:"Domain"`
		Name   string `help:"Name to alter"`
		Desc   string `help:"Description"`
	}
	R(&RoleUpdateOptions{}, "role-update", "Update role", func(s *mcclient.ClientSession, args *RoleUpdateOptions) error {
		query := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			query.Add(jsonutils.NewString(domainId), "domain_id")
		}
		rid, err := modules.RolesV3.GetId(s, args.ID, query)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		role, err := modules.RolesV3.Patch(s, rid, params)
		if err != nil {
			return err
		}
		printObject(role)
		return nil
	})
}
