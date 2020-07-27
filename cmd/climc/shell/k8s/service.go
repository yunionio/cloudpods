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

func initService() {
	cmdN := NewCmdNameFactory("service")
	svcCmd := NewShellCommands(cmdN.Do).AddR(
		NewK8sNsResourceGetCmd(cmdN, k8s.Services),
		NewK8sNsResourceDeleteCmd(cmdN, k8s.Services),
	)

	listCmd := NewCommand(
		&o.ServiceListOptions{},
		cmdN.Do("list"),
		"List Services resource",
		func(s *mcclient.ClientSession, args *o.ServiceListOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.Services.List(s, params)
			if err != nil {
				return err
			}
			PrintListResultTable(ret, k8s.Services, s)
			return nil
		},
	)

	createCmd := NewCommand(
		&o.ServiceCreateOptions{},
		cmdN.Do("create"),
		"Create service resource",
		func(s *mcclient.ClientSession, args *o.ServiceCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.Services.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
	svcCmd.AddR(listCmd, createCmd)
}
