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
	"io/ioutil"

	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		Id     []string `help:"IDs of instances to show"`
		Zone   string   `help:"Zone ID"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *aws.SRegion, args *InstanceListOptions) error {
		instances, total, e := cli.GetInstances(args.Zone, args.Id, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(instances, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type InstanceCreateOptions struct {
		NAME      string `help:"name of instance"`
		IMAGE     string `help:"image ID"`
		CPU       int    `help:"CPU count"`
		MEMORYGB  int    `help:"MemoryGB"`
		Disk      []int  `help:"Data disk sizes int GB"`
		STORAGE   string `help:"Storage type" choices:"gp2|io1|st1|sc1|standard"`
		NETWORK   string `help:"Network ID"`
		PUBLICKEY string `help:"PublicKey file path"`
	}
	shellutils.R(&InstanceCreateOptions{}, "instance-create", "Create a instance", func(cli *aws.SRegion, args *InstanceCreateOptions) error {
		content, err := ioutil.ReadFile(args.PUBLICKEY)
		if err != nil {
			return err
		}
		instance, e := cli.CreateInstanceSimple(args.NAME, args.IMAGE, args.CPU, args.MEMORYGB, args.STORAGE, args.Disk, args.NETWORK, string(content))
		if e != nil {
			return e
		}
		printObject(instance)
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

	shellutils.R(&InstanceDiskAttachOptions{}, "instance-attach-disk", "Attach a disk to instance", func(cli *aws.SRegion, args *InstanceDiskAttachOptions) error {
		err := cli.AttachDisk(args.ID, args.DISK, args.DEVICE)
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&InstanceDiskOperationOptions{}, "instance-detach-disk", "Detach a disk to instance", func(cli *aws.SRegion, args *InstanceDiskOperationOptions) error {
		err := cli.DetachDisk(args.ID, args.DISK)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceOperationOptions struct {
		ID string `help:"instance ID"`
	}
	shellutils.R(&InstanceOperationOptions{}, "instance-start", "Start a instance", func(cli *aws.SRegion, args *InstanceOperationOptions) error {
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
	shellutils.R(&InstanceStopOptions{}, "instance-stop", "Stop a instance", func(cli *aws.SRegion, args *InstanceStopOptions) error {
		err := cli.StopVM(args.ID, args.Force)
		if err != nil {
			return err
		}
		return nil
	})
	shellutils.R(&InstanceOperationOptions{}, "instance-delete", "Delete a instance", func(cli *aws.SRegion, args *InstanceOperationOptions) error {
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

	shellutils.R(&InstanceDeployOptions{}, "instance-deploy", "Deploy keypair/password to a stopped virtual server", func(cli *aws.SRegion, args *InstanceDeployOptions) error {
		err := cli.DeployVM(args.ID, args.Name, args.Password, args.Keypair, args.DeleteKeypair, args.Description)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceRebuildRootOptions struct {
		ID    string `help:"instance ID"`
		Image string `help:"Image ID"`
		Size  int    `help:"system disk size in GB"`
	}

	shellutils.R(&InstanceRebuildRootOptions{}, "instance-rebuild-root", "Reinstall virtual server system image", func(cli *aws.SRegion, args *InstanceRebuildRootOptions) error {
		ctx := context.Background()
		diskID, err := cli.ReplaceSystemDisk(ctx, args.ID, args.Image, args.Size, "", "")
		if err != nil {
			return err
		}
		fmt.Printf("New diskID is %s", diskID)
		return nil
	})

	type InstanceChangeConfigOptions struct {
		ID             string `help:"instance ID"`
		InstanceTypeId string `help:"instance type"`
		Disk           []int  `help:"Data disk sizes int GB"`
	}

	shellutils.R(&InstanceChangeConfigOptions{}, "instance-change-config", "Deploy keypair/password to a stopped virtual server", func(cli *aws.SRegion, args *InstanceChangeConfigOptions) error {
		instance, e := cli.GetInstance(args.ID)
		if e != nil {
			return e
		}

		// todo : add create disks
		err := cli.ChangeVMConfig2(instance.ZoneId, args.ID, args.InstanceTypeId, nil)
		if err != nil {
			return err
		}
		return nil
	})
}
