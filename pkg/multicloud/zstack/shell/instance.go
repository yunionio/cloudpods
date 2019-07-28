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

	"yunion.io/x/onecloud/pkg/multicloud/zstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		HostId     string
		InstanceId string
		NicId      string
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List instances", func(cli *zstack.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstances(args.HostId, args.InstanceId, args.NicId)
		if err != nil {
			return err
		}
		printList(instances, len(instances), 0, 0, []string{})
		return nil
	})

	type InstanceOperation struct {
		ID string
	}

	shellutils.R(&InstanceOperation{}, "instance-delete", "Delete instance", func(cli *zstack.SRegion, args *InstanceOperation) error {
		return cli.DeleteVM(args.ID)
	})

	shellutils.R(&InstanceOperation{}, "instance-console-password", "Show instance console password", func(cli *zstack.SRegion, args *InstanceOperation) error {
		password, err := cli.GetInstanceConsolePassword(args.ID)
		if err != nil {
			return err
		}
		fmt.Println(password)
		return nil
	})

	shellutils.R(&InstanceOperation{}, "instance-console-info", "Show instance console info", func(cli *zstack.SRegion, args *InstanceOperation) error {
		info, err := cli.GetInstanceConsoleInfo(args.ID)
		if err != nil {
			return err
		}
		printObject(info)
		return nil
	})

	shellutils.R(&InstanceOperation{}, "instance-start", "Start instance", func(cli *zstack.SRegion, args *InstanceOperation) error {
		return cli.StartVM(args.ID)
	})

	shellutils.R(&InstanceOperation{}, "instance-boot-order", "Show instance boot order", func(cli *zstack.SRegion, args *InstanceOperation) error {
		fmt.Println(cli.GetBootOrder(args.ID))
		return nil
	})

	type InstanceStopOption struct {
		ID      string
		IsForce bool
	}

	shellutils.R(&InstanceStopOption{}, "instance-stop", "Start instance", func(cli *zstack.SRegion, args *InstanceStopOption) error {
		return cli.StopVM(args.ID, args.IsForce)
	})

	type InstanceSecgroupOption struct {
		ID       string
		SECGRPID string
	}

	shellutils.R(&InstanceSecgroupOption{}, "instance-assign-secgroup", "Assign secgroup for a instance", func(cli *zstack.SRegion, args *InstanceSecgroupOption) error {
		return cli.AssignSecurityGroup(args.ID, args.SECGRPID)
	})

	shellutils.R(&InstanceSecgroupOption{}, "instance-revoke-secgroup", "Assign secgroup for a instance", func(cli *zstack.SRegion, args *InstanceSecgroupOption) error {
		return cli.RevokeSecurityGroup(args.ID, args.SECGRPID)
	})

}
