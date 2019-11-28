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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		ZONE       string
		MaxResults int
		PageToken  string
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List instances", func(cli *google.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstances(args.ZONE, args.MaxResults, args.PageToken)
		if err != nil {
			return err
		}
		printList(instances, 0, 0, 0, nil)
		return nil
	})

	type InstanceIdOptions struct {
		ID string
	}
	shellutils.R(&InstanceIdOptions{}, "instance-show", "Show instance", func(cli *google.SRegion, args *InstanceIdOptions) error {
		instance, err := cli.GetInstance(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	shellutils.R(&InstanceIdOptions{}, "instance-delete", "Delete instance", func(cli *google.SRegion, args *InstanceIdOptions) error {
		return cli.Delete(args.ID)
	})

	shellutils.R(&InstanceIdOptions{}, "instance-start", "Start instance", func(cli *google.SRegion, args *InstanceIdOptions) error {
		return cli.StartInstance(args.ID)
	})

	shellutils.R(&InstanceIdOptions{}, "instance-stop", "Stop instance", func(cli *google.SRegion, args *InstanceIdOptions) error {
		return cli.StopInstance(args.ID)
	})

	shellutils.R(&InstanceIdOptions{}, "instance-reset", "Reset instance", func(cli *google.SRegion, args *InstanceIdOptions) error {
		return cli.ResetInstance(args.ID)
	})

	type InstanceEipOptions struct {
		ID  string
		EIP string `help:"eip address"`
	}

	shellutils.R(&InstanceEipOptions{}, "instance-dissociate-eip", "Dissociate instance eip", func(cli *google.SRegion, args *InstanceEipOptions) error {
		return cli.DissociateInstanceEip(args.ID, args.EIP)
	})

	shellutils.R(&InstanceEipOptions{}, "instance-associate-eip", "Associate instance eip", func(cli *google.SRegion, args *InstanceEipOptions) error {
		return cli.AssociateInstanceEip(args.ID, args.EIP)
	})

	type InstanceDetachDiskOptions struct {
		ID         string
		DeviceName string
	}

	shellutils.R(&InstanceDetachDiskOptions{}, "instance-detach-disk", "Detach instance disk", func(cli *google.SRegion, args *InstanceDetachDiskOptions) error {
		return cli.DetachDisk(args.ID, args.DeviceName)
	})

	type InstanceSetPublicKeyOptions struct {
		ID        string
		PublicKey string
	}

	shellutils.R(&InstanceSetPublicKeyOptions{}, "instance-set-publickey", "Set instance public key", func(cli *google.SRegion, args *InstanceSetPublicKeyOptions) error {
		instance, err := cli.GetInstance(args.ID)
		if err != nil {
			return errors.Wrap(err, "cli.GetInstance")
		}
		items := []google.SMetadataItem{}
		for _, item := range instance.Metadata.Items {
			if item.Key != google.METADATA_SSH_KEYS {
				items = append(items, item)
			}
		}
		if len(args.PublicKey) > 0 {
			items = append(items, google.SMetadataItem{Key: google.METADATA_SSH_KEYS, Value: "root:" + args.PublicKey})
		}
		instance.Metadata.Items = items
		return cli.SetMetadata(args.ID, instance.Metadata)
	})

	type InstanceAttachDiskOptions struct {
		ID   string
		DISK string
		Boot bool
	}

	type InstanceSerialOutput struct {
		ID   string
		PORT int
	}

	shellutils.R(&InstanceSerialOutput{}, "instance-serial-output", "Get instance serial output", func(cli *google.SRegion, args *InstanceSerialOutput) error {
		content, err := cli.GetSerialPortOutput(args.ID, args.PORT)
		if err != nil {
			return err
		}
		fmt.Printf("content: %s\n", content)
		return nil
	})

	shellutils.R(&InstanceAttachDiskOptions{}, "instance-attach-disk", "Attach instance disk", func(cli *google.SRegion, args *InstanceAttachDiskOptions) error {
		return cli.AttachDisk(args.ID, args.DISK, args.Boot)
	})

	type InstanceRebuildRootOptions struct {
		ID         string
		IMAGE      string
		DiskSizeGb int
	}

	shellutils.R(&InstanceRebuildRootOptions{}, "instance-rebuild-root", "Rebuild instance root", func(cli *google.SRegion, args *InstanceRebuildRootOptions) error {
		diskId, err := cli.RebuildRoot(args.ID, args.IMAGE, args.DiskSizeGb)
		if err != nil {
			return err
		}
		fmt.Println(diskId)
		return nil
	})

	type InstanceChangeConfigOptions struct {
		ID           string
		ZONE         string
		InstanceType string
		Cpu          int
		MemoryMb     int
	}

	shellutils.R(&InstanceChangeConfigOptions{}, "instance-change-config", "Change instance config", func(cli *google.SRegion, args *InstanceChangeConfigOptions) error {
		return cli.ChangeInstanceConfig(args.ID, args.ZONE, args.InstanceType, args.Cpu, args.MemoryMb)
	})

	type InstanceCreateOptions struct {
		NAME         string
		ZONE         string
		IMAGE        string
		InstanceType string
		Cpu          int
		MemoryMb     int
		NETWORK      string
		IpAddr       string
		Desc         string
		DISKS        []string `nargs:"+"`
	}

	shellutils.R(&InstanceCreateOptions{}, "instance-create", "Create instance", func(cli *google.SRegion, args *InstanceCreateOptions) error {
		instance, err := cli.CreateInstance(args.ZONE, args.NAME, args.Desc, args.InstanceType, args.Cpu, args.MemoryMb, args.NETWORK, args.IpAddr, args.IMAGE, args.DISKS)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

}
