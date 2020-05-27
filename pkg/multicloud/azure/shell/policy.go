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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type PolicyListOptions struct {
	}
	shellutils.R(&PolicyListOptions{}, "policy-definition-list", "List policy definitions", func(cli *azure.SRegion, args *PolicyListOptions) error {
		definitions, err := cli.GetClient().GetPolicyDefinitions()
		if err != nil {
			return err
		}
		printList(definitions, len(definitions), 0, 0, []string{})
		return nil
	})

	type PolicyAssignmentListOptions struct {
		DefinitionId string
	}

	shellutils.R(&PolicyAssignmentListOptions{}, "policy-assignment-list", "List policy assignment", func(cli *azure.SRegion, args *PolicyAssignmentListOptions) error {
		assignments, err := cli.GetClient().GetPolicyAssignments(args.DefinitionId)
		if err != nil {
			return err
		}
		printList(assignments, len(assignments), 0, 0, []string{})
		return nil
	})

	type PolicyIdOptions struct {
		ID string
	}

	shellutils.R(&PolicyIdOptions{}, "policy-definition-show", "Show policy definition", func(cli *azure.SRegion, args *PolicyIdOptions) error {
		definition, err := cli.GetClient().GetPolicyDefinition(args.ID)
		if err != nil {
			return err
		}
		printObject(definition)
		return nil
	})

}
