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
	huawei "yunion.io/x/onecloud/pkg/multicloud/huaweistack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type EnterpriseProjectListOptions struct {
	}
	shellutils.R(&EnterpriseProjectListOptions{}, "enterprise-project-list", "List enterprise projects", func(cli *huawei.SRegion, args *EnterpriseProjectListOptions) error {
		projects, err := cli.GetClient().GetEnterpriseProjects()
		if err != nil {
			return err
		}
		printList(projects, 0, 0, 0, nil)
		return nil
	})

	type EnterpriseProjectCreateOptions struct {
		NAME string
		Desc string
	}

	shellutils.R(&EnterpriseProjectCreateOptions{}, "enterprise-project-create", "Create enterprise project", func(cli *huawei.SRegion, args *EnterpriseProjectCreateOptions) error {
		project, err := cli.GetClient().CreateExterpriseProject(args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(project)
		return nil
	})

}
