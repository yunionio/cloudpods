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

package drivers

import (
	"fmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	_ "yunion.io/x/onecloud/pkg/baremetal/utils/raid/hpssactl"
	_ "yunion.io/x/onecloud/pkg/baremetal/utils/raid/megactl"
	_ "yunion.io/x/onecloud/pkg/baremetal/utils/raid/mvcli"
	_ "yunion.io/x/onecloud/pkg/baremetal/utils/raid/sas2iru"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

func GetDriver(name string, term *ssh.Client) raid.IRaidDriver {
	factory := raid.RaidDrivers[name]
	if factory == nil {
		return nil
	}
	return factory(term)
}

func GetDrivers(term *ssh.Client) []raid.IRaidDriver {
	ret := []raid.IRaidDriver{}
	for _, factory := range raid.RaidDrivers {
		ret = append(ret, factory(term))
	}
	return ret
}

func BuildRaid(driver raid.IRaidDriver, confs []*api.BaremetalDiskConfig, adapterIdx int) error {
	if err := driver.PreBuildRaid(confs, adapterIdx); err != nil {
		return fmt.Errorf("PreBuildRaid: %v", err)
	}
	var adapter raid.IRaidAdapter
	for _, tmp := range driver.GetAdapters() {
		if tmp.GetIndex() == adapterIdx {
			adapter = tmp
			break
		}
	}
	if adapter == nil {
		return fmt.Errorf("Not found adapter by index %d", adapterIdx)
	}
	if err := buildRaid(adapter, confs); err != nil {
		return fmt.Errorf("Driver %s, adapter %d build raid: %v", driver.GetName(), adapterIdx, err)
	}
	return nil
}

func buildRaid(adapter raid.IRaidAdapter, confs []*api.BaremetalDiskConfig) error {
	if err := adapter.PreBuildRaid(confs); err != nil {
		return fmt.Errorf("PreBuildRaid: %v", err)
	}
	if err := adapter.RemoveLogicVolumes(); err != nil {
		return fmt.Errorf("RemoveLogicVolumes: %v", err)
	}
	devs := adapter.GetDevices()
	if len(devs) == 0 {
		// no disk to build
		return nil
	}

	var selected []*baremetal.BaremetalStorage
	var nonDisks []*baremetal.BaremetalStorage
	left := devs

	for _, conf := range confs {
		selected, left = baremetal.RetrieveStorages(conf, left)
		if len(selected) == 0 {
			return fmt.Errorf("No enough disks for config %#v", conf)
		}
		var err error
		switch conf.Conf {
		case baremetal.DISK_CONF_RAID5:
			err = adapter.BuildRaid5(selected, conf)
		case baremetal.DISK_CONF_RAID10:
			err = adapter.BuildRaid10(selected, conf)
		case baremetal.DISK_CONF_NONE:
			nonDisks = append(nonDisks, selected...)
		case baremetal.DISK_CONF_RAID0:
			err = adapter.BuildRaid0(selected, conf)
		case baremetal.DISK_CONF_RAID1:
			err = adapter.BuildRaid1(selected, conf)
		default:
			return fmt.Errorf("Unknown raid config %s", conf.Conf)
		}
		if err != nil {
			return fmt.Errorf("Build raid %s: %v", conf.Conf, err)
		}
	}
	if len(nonDisks) > 0 {
		if err := adapter.BuildNoneRaid(nonDisks); err != nil {
			return fmt.Errorf("Build raw disks: %v", err)
		}
	}
	return nil
}
