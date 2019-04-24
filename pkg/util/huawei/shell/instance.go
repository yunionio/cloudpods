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
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *huawei.SRegion, args *InstanceListOptions) error {
		instances, e := cli.GetInstances()
		if e != nil {
			return e
		}
		printList(instances, 0, 0, 0, nil)
		return nil
	})

	type InstanceDiskOperationOptions struct {
		ID   string `help:"instance ID"`
		DISK string `help:"disk ID"`
	}

	type InstanceDiskAttachOptions struct {
		ID     string `help:"instance ID"`
		DISK   string `help:"disk ID"`
		DEVICE string `help:"disk device name. eg. /dev/sdb"`
	}

	shellutils.R(&InstanceDiskAttachOptions{}, "instance-attach-disk", "Attach a disk to instance", func(cli *huawei.SRegion, args *InstanceDiskAttachOptions) error {
		err := cli.AttachDisk(args.ID, args.DISK, args.DEVICE)
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&InstanceDiskOperationOptions{}, "instance-detach-disk", "Detach a disk to instance", func(cli *huawei.SRegion, args *InstanceDiskOperationOptions) error {
		err := cli.DetachDisk(args.ID, args.DISK)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceOperationOptions struct {
		ID string `help:"instance ID"`
	}
	shellutils.R(&InstanceOperationOptions{}, "instance-start", "Start a instance", func(cli *huawei.SRegion, args *InstanceOperationOptions) error {
		err := cli.StartVM(args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceStopOptions struct {
		ID    string `help:"instance ID"`
		Force bool   `help:"Force stop instance"`
	}
	shellutils.R(&InstanceStopOptions{}, "instance-stop", "Stop a instance", func(cli *huawei.SRegion, args *InstanceStopOptions) error {
		err := cli.StopVM(args.ID, args.Force)
		if err != nil {
			return err
		}
		return nil
	})
	shellutils.R(&InstanceOperationOptions{}, "instance-delete", "Delete a instance", func(cli *huawei.SRegion, args *InstanceOperationOptions) error {
		err := cli.DeleteVM(args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	/*
		server-change-config 更改系统配置
		server-reset
	*/
	type InstanceDeployOptions struct {
		ID            string `help:"instance ID"`
		Name          string `help:"new instance name"`
		Hostname      string `help:"new hostname"`
		Keypair       string `help:"Keypair Name"`
		DeleteKeypair bool   `help:"Remove SSH keypair"`
		Password      string `help:"new password"`
		// ResetPassword bool   `help:"Force reset password"`
		Description string `help:"new instances description"`
	}

	shellutils.R(&InstanceDeployOptions{}, "instance-deploy", "Deploy keypair/password to a stopped virtual server", func(cli *huawei.SRegion, args *InstanceDeployOptions) error {
		err := cli.DeployVM(args.ID, args.Name, args.Password, args.Keypair, args.DeleteKeypair, args.Description)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceRebuildRootOptions struct {
		ID        string `help:"instance ID"`
		Image     string `help:"Image ID"`
		Password  string `help:"admin password"`
		PublicKey string `help:"public key name"`
	}

	shellutils.R(&InstanceRebuildRootOptions{}, "instance-rebuild-root", "Reinstall virtual server system image", func(cli *huawei.SRegion, args *InstanceRebuildRootOptions) error {
		ctx := context.Background()
		jobId, err := cli.ChangeRoot(ctx, args.ID, args.Image, args.Password, args.PublicKey)
		if err != nil {
			return err
		}
		fmt.Printf("ChangeRoot jobID is %s", jobId)
		return nil
	})

	type InstanceChangeConfigOptions struct {
		ID             string `help:"instance ID"`
		InstanceTypeId string `help:"instance type"`
		Disk           []int  `help:"Data disk sizes int GB"`
	}

	shellutils.R(&InstanceChangeConfigOptions{}, "instance-change-config", "Deploy keypair/password to a stopped virtual server", func(cli *huawei.SRegion, args *InstanceChangeConfigOptions) error {
		instance, e := cli.GetInstanceByID(args.ID)
		if e != nil {
			return e
		}

		// todo : add create disks
		err := cli.ChangeVMConfig2(instance.GetId(), args.ID, args.InstanceTypeId, nil)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceOrderUnsubscribeOptions struct {
		ID     string `help:"instance ID"`
		DOMAIN string `help:"domain ID"`
	}

	shellutils.R(&InstanceOrderUnsubscribeOptions{}, "instance-order-unsubscribe", "Unsubscribe a prepaid server", func(cli *huawei.SRegion, args *InstanceOrderUnsubscribeOptions) error {
		instance, e := cli.GetInstanceByID(args.ID)
		if e != nil {
			return e
		}

		_, err := cli.UnsubscribeInstance(instance.GetId(), args.DOMAIN)
		if err != nil {
			return err
		}
		return nil
	})
}
