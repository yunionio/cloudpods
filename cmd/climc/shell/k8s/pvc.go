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

func initPVC() {
	cmdN := NewCmdNameFactory("pvc")
	pvcCmd := NewShellCommands(cmdN.Do).AddR(
		NewK8sNsResourceGetCmd(cmdN, k8s.PersistentVolumeClaims),
		NewK8sNsResourceDeleteCmd(cmdN, k8s.PersistentVolumeClaims),
	)
	listCmd := NewCommand(
		&o.PVCListOptions{},
		cmdN.Do("list"),
		"List PersistentVolumeClaims resource",
		func(s *mcclient.ClientSession, args *o.PVCListOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.PersistentVolumeClaims.List(s, params)
			if err != nil {
				return err
			}
			PrintListResultTable(ret, k8s.PersistentVolumeClaims, s)
			return nil
		},
	)
	createCmd := NewCommand(
		&o.PVCCreateOptions{},
		cmdN.Do("create"),
		"Create PersistentVolumeClaims resource",
		func(s *mcclient.ClientSession, args *o.PVCCreateOptions) error {
			ret, err := k8s.PersistentVolumeClaims.Create(s, args.Params())
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)
	pvcCmd.AddR(listCmd, createCmd)
}
