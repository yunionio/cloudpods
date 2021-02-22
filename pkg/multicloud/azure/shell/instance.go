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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		Classic   bool `help:"List classic instance"`
		ScaleSets bool `help:"List Scale Sets instance"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *azure.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstances()
		if err != nil {
			return err
		}
		printList(instances, len(instances), 0, 0, []string{})
		return nil
	})

	shellutils.R(&InstanceListOptions{}, "classic-instance-list", "List classic instance", func(cli *azure.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetClassicInstances()
		if err != nil {
			return err
		}
		printList(instances, len(instances), 0, 0, []string{})
		return nil
	})

	type SClassicInstacneIdOptions struct {
		ID string
	}

	shellutils.R(&SClassicInstacneIdOptions{}, "classic-instance-disk-list", "List classic instance disks", func(cli *azure.SRegion, args *SClassicInstacneIdOptions) error {
		disks, err := cli.GetClassicInstanceDisks(args.ID)
		if err != nil {
			return err
		}
		printList(disks, len(disks), 0, 0, []string{})
		return nil
	})

	shellutils.R(&InstanceListOptions{}, "instance-scaleset-list", "List classic instance", func(cli *azure.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstanceScaleSets()
		if err != nil {
			return err
		}
		printList(instances, len(instances), 0, 0, []string{})
		return nil
	})

	type InstanceSizeListOptions struct {
	}

	shellutils.R(&InstanceSizeListOptions{}, "instance-size-list", "List intances", func(cli *azure.SRegion, args *InstanceSizeListOptions) error {
		vmSizes, err := cli.ListVmSizes()
		if err != nil {
			return err
		}
		printList(vmSizes, 0, 0, 0, nil)
		return nil
	})
	shellutils.R(&InstanceSizeListOptions{}, "resource-sku-list", "List resource sku", func(cli *azure.SRegion, args *InstanceSizeListOptions) error {
		skus, err := cli.GetClient().ListResourceSkus()
		if err != nil {
			return err
		}
		printList(skus, len(skus), 0, 0, []string{})
		return nil
	})

	type InstanceCreateOptions struct {
		NAME          string `help:"Name of instance"`
		IMAGE         string `help:"image ID"`
		CPU           int    `help:"CPU count"`
		MEMORYMB      int    `help:"MemoryMb"`
		InstanceType  string `help:"Instance Type"`
		SYSDISKSIZEGB int    `help:"System Disk Size"`
		Disk          []int  `help:"Data disk sizes int GB"`
		STORAGE       string `help:"Storage type"`
		NETWORK       string `help:"Network ID"`
		PASSWD        string `help:"password"`
		PublicKey     string `help:"PublicKey"`
		OsType        string `help:"Operation system type" choices:"Linux|Windows"`
	}
	shellutils.R(&InstanceCreateOptions{}, "instance-create", "Create a instance", func(cli *azure.SRegion, args *InstanceCreateOptions) error {
		instance, e := cli.CreateInstanceSimple(args.NAME, args.IMAGE, args.OsType, args.CPU, args.MEMORYMB, args.SYSDISKSIZEGB, args.STORAGE, args.Disk, args.NETWORK, args.PASSWD, args.PublicKey)
		if e != nil {
			return e
		}
		printObject(instance)
		return nil
	})

	type InstanceOptions struct {
		ID string `help:"Instance ID"`
	}
	shellutils.R(&InstanceOptions{}, "instance-show", "Show intance detail", func(cli *azure.SRegion, args *InstanceOptions) error {
		if instance, err := cli.GetInstance(args.ID); err != nil {
			return err
		} else {
			printObject(instance)
			return nil
		}
	})

	shellutils.R(&InstanceOptions{}, "instance-start", "Start intance", func(cli *azure.SRegion, args *InstanceOptions) error {
		return cli.StartVM(args.ID)
	})

	shellutils.R(&InstanceOptions{}, "instance-delete", "Delete intance", func(cli *azure.SRegion, args *InstanceOptions) error {
		return cli.DeleteVM(args.ID)
	})

	shellutils.R(&InstanceOptions{}, "instance-stop", "Stop intance", func(cli *azure.SRegion, args *InstanceOptions) error {
		return cli.StopVM(args.ID, true)
	})

	type InstanceRebuildOptions struct {
		ID        string `help:"Instance ID"`
		CPU       int    `help:"Instance CPU core"`
		MEMORYMB  int    `help:"Instance Memory MB"`
		IMAGE     string `help:"Image ID"`
		Password  string `help:"pasword"`
		PublicKey string `help:"Public Key"`
		Size      int    `help:"system disk size in GB"`
	}
	shellutils.R(&InstanceRebuildOptions{}, "instance-rebuild-root", "Reinstall virtual server system image", func(cli *azure.SRegion, args *InstanceRebuildOptions) error {
		instance, err := cli.GetInstance(args.ID)
		if err != nil {
			return err
		}
		diskId, err := cli.ReplaceSystemDisk(instance, args.CPU, args.MEMORYMB, args.IMAGE, args.Password, args.PublicKey, args.Size)
		if err != nil {
			return err
		}
		fmt.Printf("New diskId is %s", diskId)
		return nil
	})

	type InstanceDiskOptions struct {
		ID   string `help:"Instance ID"`
		DISK string `help:"Disk ID"`
	}
	shellutils.R(&InstanceDiskOptions{}, "instance-attach-disk", "Attach a disk to intance", func(cli *azure.SRegion, args *InstanceDiskOptions) error {
		return cli.AttachDisk(args.ID, args.DISK)
	})

	shellutils.R(&InstanceDiskOptions{}, "instance-detach-disk", "Attach a disk to intance", func(cli *azure.SRegion, args *InstanceDiskOptions) error {
		return cli.DetachDisk(args.ID, args.DISK)
	})

	type InstanceConfigOptions struct {
		ID            string `help:"Instance ID"`
		INSTANCE_TYPE string `help:"Instance Vm Size"`
	}

	shellutils.R(&InstanceConfigOptions{}, "instance-change-config", "Attach a disk to intance", func(cli *azure.SRegion, args *InstanceConfigOptions) error {
		return cli.ChangeConfig(args.ID, args.INSTANCE_TYPE)
	})

	type InstanceDeployOptions struct {
		ID        string `help:"Instance ID"`
		OsType    string `help:"Instance Os Type" choices:"Linux|Windows" default:"Linux"`
		Password  string `help:"Password for instance"`
		PublicKey string `helo:"Deploy ssh_key for instance"`
	}

	shellutils.R(&InstanceDeployOptions{}, "instance-reset-password", "Reset intance password", func(cli *azure.SRegion, args *InstanceDeployOptions) error {
		return cli.DeployVM(context.Background(), args.ID, args.OsType, "", args.Password, args.PublicKey, false, "")
	})

	type InstanceSecurityGroupOptions struct {
		ID            string `help:"Instance ID"`
		SecurityGroup string `help:"Security Group ID or Name"`
	}

	shellutils.R(&InstanceSecurityGroupOptions{}, "instance-set-secgrp", "Attach a disk to intance", func(cli *azure.SRegion, args *InstanceSecurityGroupOptions) error {
		return cli.SetSecurityGroup(args.ID, args.SecurityGroup)
	})

	type InstanceSaveImageOptions struct {
		DISK_ID    string `help:"Instance Os Disk ID"`
		IMAGE_NAME string `help:"Image name"`
		Notes      string `hlep:"Image desc"`
		OsType     string `help:"Os Type" choices:"Linux|Windows" default:"Linux"`
	}
	shellutils.R(&InstanceSaveImageOptions{}, "instance-save-image", "Save instance to image", func(cli *azure.SRegion, args *InstanceSaveImageOptions) error {
		opts := cloudprovider.SaveImageOptions{
			Name:  args.IMAGE_NAME,
			Notes: args.Notes,
		}
		image, err := cli.SaveImage(args.OsType, args.DISK_ID, &opts)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

	shellutils.R(&InstanceOptions{}, "instance-get-tags", "get intance tags", func(cli *azure.SRegion, args *InstanceOptions) error {
		tags, err := cli.GetClient().GetTags(args.ID)
		if err != nil {
			return err
		}
		printObject(tags)
		return nil
	})

	type InstanceSetTagsOptions struct {
		ID   string `help:"Instance ID"`
		Tags []string
	}
	shellutils.R(&InstanceSetTagsOptions{}, "instance-set-tags", "set intance metadata", func(cli *azure.SRegion, args *InstanceSetTagsOptions) error {
		tags := map[string]string{}
		for i := range args.Tags {
			splited := strings.Split(args.Tags[i], "=")
			if len(splited) == 2 {
				tags[splited[0]] = splited[1]
			}
		}
		result, err := cli.GetClient().SetTags(args.ID, tags)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
