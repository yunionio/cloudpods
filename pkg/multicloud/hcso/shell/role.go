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
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	cloudid_api "yunion.io/x/onecloud/pkg/apis/cloudid"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	huawei "yunion.io/x/onecloud/pkg/multicloud/hcso"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type RoleListOptions struct {
		DomainId string
		Name     string
	}
	shellutils.R(&RoleListOptions{}, "cloud-policy-list", "List cloudpolicy", func(cli *huawei.SRegion, args *RoleListOptions) error {
		roles, err := cli.GetClient().GetRoles(args.DomainId, args.Name)
		if err != nil {
			return err
		}
		printList(roles, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&RoleListOptions{}, "cloud-custom-policy-list", "List cloudpolicy", func(cli *huawei.SRegion, args *RoleListOptions) error {
		roles, err := cli.GetClient().GetCustomRoles()
		if err != nil {
			return err
		}
		printList(roles, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&RoleListOptions{}, "cloud-policy-export", "Export cloudpolicy", func(cli *huawei.SRegion, args *RoleListOptions) error {
		roles, err := cli.GetClient().GetRoles(args.DomainId, args.Name)
		if err != nil {
			return err
		}
		type sRule struct {
			Name        string
			Id          string
			ExternalId  string
			CloudEnv    string
			Provider    string
			Description string
			Document    jsonutils.JSONDict
			PolicyType  string
			Status      string
		}
		idMap := map[string]bool{}
		ret := []sRule{}
		for i := range roles {
			if _, ok := idMap[roles[i].DisplayName]; ok {
				log.Errorf("duplicate id: %s", roles[i].DisplayName)
				continue
			}
			idMap[roles[i].DisplayName] = true
			ret = append(ret, sRule{
				Name:        roles[i].DisplayName,
				Id:          roles[i].Id,
				ExternalId:  roles[i].DisplayName,
				CloudEnv:    api.CLOUD_PROVIDER_HCSO,
				Provider:    api.CLOUD_PROVIDER_HCSO,
				Description: roles[i].DescriptionCn,
				Document:    roles[i].Policy,
				PolicyType:  cloudid_api.CLOUD_POLICY_TYPE_SYSTEM,
				Status:      cloudid_api.CLOUD_POLICY_STATUS_AVAILABLE,
			})
		}
		return ioutil.WriteFile(fmt.Sprintf("%s.json", api.CLOUD_PROVIDER_HCSO), []byte(jsonutils.Marshal(ret).PrettyString()), 0644)
	})
}
