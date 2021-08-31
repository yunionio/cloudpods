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
	"strings"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	huawei "yunion.io/x/onecloud/pkg/multicloud/hcso"
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
		ID            string `help:"instance ID"`
		UserId        string `help:"instance user ID"`
		Image         string `help:"Image ID"`
		Password      string `help:"admin password"`
		PublicKeyName string `help:"public key name"`
		UserData      string `help:"cloud-init user data"`
	}

	shellutils.R(&InstanceRebuildRootOptions{}, "instance-rebuild-root", "Reinstall virtual server system image", func(cli *huawei.SRegion, args *InstanceRebuildRootOptions) error {
		ctx := context.Background()
		jobId, err := cli.ChangeRoot(ctx, args.UserId, args.ID, args.Image, args.Password, args.PublicKeyName, args.UserData)
		if err != nil {
			return err
		}
		fmt.Printf("ChangeRoot jobID is %s", jobId)
		return nil
	})

	type InstanceChangeConfigOptions struct {
		ID           string `help:"instance ID"`
		InstanceType string `help:"instance type"`
	}

	shellutils.R(&InstanceChangeConfigOptions{}, "instance-change-config", "Deploy keypair/password to a stopped virtual server", func(cli *huawei.SRegion, args *InstanceChangeConfigOptions) error {
		err := cli.ChangeVMConfig(args.ID, args.InstanceType)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceOrderUnsubscribeOptions struct {
		ID     string `help:"instance ID"`
		DOMAIN string `help:"domain ID"`
	}

	type InstanceSaveImageOptions struct {
		ID         string `help:"Instance ID"`
		IMAGE_NAME string `help:"Image name"`
		Notes      string `hlep:"Image desc"`
	}
	shellutils.R(&InstanceSaveImageOptions{}, "instance-save-image", "Save instance to image", func(cli *huawei.SRegion, args *InstanceSaveImageOptions) error {
		opts := cloudprovider.SaveImageOptions{
			Name:  args.IMAGE_NAME,
			Notes: args.Notes,
		}
		image, err := cli.SaveImage(args.ID, &opts)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

	type InstanceSetTagsOptions struct {
		ID   string `help:"Instance ID"`
		Tags []string
	}
	shellutils.R(&InstanceSetTagsOptions{}, "instance-set-tags", "get intance metadata", func(cli *huawei.SRegion, args *InstanceSetTagsOptions) error {
		tags := map[string]string{}
		for i := range args.Tags {
			splited := strings.Split(args.Tags[i], "=")
			if len(splited) == 2 {
				tags[splited[0]] = splited[1]
			}
		}
		err := cli.CreateServerTags(args.ID, tags)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceDelTagsOptions struct {
		ID   string `help:"Instance ID"`
		Tags []string
	}
	shellutils.R(&InstanceDelTagsOptions{}, "instance-del-tags", "del intance metadata", func(cli *huawei.SRegion, args *InstanceDelTagsOptions) error {

		err := cli.DeleteServerTags(args.ID, args.Tags)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceUpdateNameOptions struct {
		ID   string `help:"Instance ID"`
		Name string
	}
	shellutils.R(&InstanceUpdateNameOptions{}, "instance-set-name", "set intance name", func(cli *huawei.SRegion, args *InstanceUpdateNameOptions) error {

		err := cli.UpdateVM(args.ID, args.Name)
		if err != nil {
			return err
		}
		return nil
	})
}
