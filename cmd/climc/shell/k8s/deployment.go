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

package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initDeployment() {
	cmd := initK8sNamespaceResource("deployment", k8s.Deployments)
	cmdN := cmd.CommandNameFactory

	createCmd := NewCommand(
		&o.DeploymentCreateOptions{},
		cmdN("create"),
		"Create deployment resource",
		func(s *mcclient.ClientSession, args *o.DeploymentCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.Deployments.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	updateCmd := NewCommand(
		&o.DeploymentUpdateOptions{},
		cmdN("update"),
		"Update deployment resource",
		func(s *mcclient.ClientSession, args *o.DeploymentUpdateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.Deployments.Update(s, args.NAME, params)
			if err != nil {
				return err
			}
			printObjectYAML(ret)
			return nil
		})

	cmd.AddR(createCmd, updateCmd)
}
