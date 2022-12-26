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
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/onecloud/pkg/mcclient/cloudpods"
)

func init() {
	type InstanceListOptions struct {
		HostId string
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List instances", func(cli *cloudpods.SRegion, args *InstanceListOptions) error {
		instances, err := cli.GetInstances(args.HostId)
		if err != nil {
			return err
		}
		printList(instances, 0, 0, 0, nil)
		return nil
	})

	type InstanceIdOptions struct {
		ID string
	}

	shellutils.R(&InstanceIdOptions{}, "instance-show", "Show instance", func(cli *cloudpods.SRegion, args *InstanceIdOptions) error {
		instance, err := cli.GetInstance(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

	shellutils.R(&InstanceIdOptions{}, "instance-vnc", "Show instance vnc", func(cli *cloudpods.SRegion, args *InstanceIdOptions) error {
		vnc, err := cli.GetInstanceVnc(args.ID, "")
		if err != nil {
			return err
		}
		printObject(vnc)
		return nil
	})

	type InstanceSaveImageOptions struct {
		ID   string
		NAME string
		Note string
	}

	shellutils.R(&InstanceSaveImageOptions{}, "instance-save-image", "Save instance image", func(cli *cloudpods.SRegion, args *InstanceSaveImageOptions) error {
		image, err := cli.SaveImage(args.ID, args.NAME, args.Note)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

	shellutils.R(&cloudprovider.MetricListOptions{}, "instance-monitor-list", "Instance Monitor List", func(cli *cloudpods.SRegion, opts *cloudprovider.MetricListOptions) error {
		metrics, err := cli.GetClient().GetMetrics(opts)
		if err != nil {
			return err
		}
		printList(metrics, len(metrics), 0, 0, []string{})
		return nil
	})
}
