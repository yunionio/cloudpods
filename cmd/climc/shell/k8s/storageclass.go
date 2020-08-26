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

func initStorageClass() {
	cmdN := NewCmdNameFactory("storageclass")
	scCmd := initK8sClusterResource("storageclass", k8s.Storageclass)

	setDefaultCmd := NewCommand(
		&o.ClusterResourceBaseOptions{},
		cmdN.Do("set-default"),
		"Set storageclass as default",
		func(s *mcclient.ClientSession, args *o.ClusterResourceBaseOptions) error {
			ret, err := k8s.Storageclass.PerformAction(s, args.NAME, "set-default", args.Params())
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)

	scCmd.AddR(setDefaultCmd)

	addStorageClassCephCSI(scCmd)
}

func addStorageClassCephCSI(cmd *ShellCommands) {
	rbdN := NewCmdNameFactory("storageclass-ceph-csi-rbd")
	rbdCreateCmd := NewCommand(
		&o.StorageClassCephCSIRBDCreateOptions{},
		rbdN.Do("create"),
		"Create ceph csi rbd",
		func(s *mcclient.ClientSession, args *o.StorageClassCephCSIRBDCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.Storageclass.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)
	testConnCmd := NewCommand(
		&o.StorageClassCephCSIRBDTestOptions{},
		rbdN.Do("connection-test"),
		"Test storageclass connection",
		func(s *mcclient.ClientSession, args *o.StorageClassCephCSIRBDTestOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := k8s.Storageclass.PerformClassAction(s, "connection-test", params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)

	cmd.AddR(rbdCreateCmd, testConnCmd)
}
