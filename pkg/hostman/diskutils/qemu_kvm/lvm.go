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

package qemu_kvm

import (
	"encoding/json"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/guestfs/kvmpart"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func (d *LocalDiskDriver) setupLVMS() {
	for _, part := range d.partitions {
		vg, err := d.findVg(part.GetPartDev())
		if err != nil {
			log.Infof("failed find vg %s", err)
			continue
		}
		if vg == nil {
			continue
		}

		log.Infof("found vg %s from %s", vg.VgName, part.GetPartDev())
		err = d.vgActive(vg.VgName)
		if err != nil {
			log.Infof("failed active vg %s: %s", vg.VgName, err)
			continue
		}
		lvs, err := d.getVgLvs(vg.VgName)
		if err != nil {
			log.Infof("failed get vg lvs %s: %s", vg.VgName, err)
			continue
		}
		log.Infof("found lvs %v from vg %s", lvs, vg.VgName)
		for _, lv := range lvs {
			lvmpart := kvmpart.NewKVMGuestDiskPartition(lv.LvPath, "", true)
			d.lvmPartitions = append(d.lvmPartitions, lvmpart)
			log.Infof("found lvm part dev %v", lvmpart.GetPartDev())
		}
	}
}

func (d *LocalDiskDriver) vgActive(vgname string) error {
	out, err := procutils.NewCommand("vgchange", "-ay", vgname).Output()
	if err != nil {
		return errors.Wrapf(err, "vgchange -ay %s %s", vgname, out)
	}
	return nil
}

type LvProps struct {
	LvName string
	LvPath string
}

type LvNames struct {
	Report []struct {
		LV []struct {
			LVName string `json:"lv_name"`
			LVPath string `json:"lv_path"`
		} `json:"lv"`
	} `json:"report"`
}

func (d *LocalDiskDriver) getVgLvs(vg string) ([]LvProps, error) {
	cmd := fmt.Sprintf("lvs --reportformat json -o lv_name,lv_path %s 2>/dev/null", vg)
	out, err := procutils.NewCommand("sh", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrap(err, "find vg lvs")
	}
	var res LvNames
	err = json.Unmarshal(out, &res)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal lvs")
	}
	if len(res.Report) != 1 {
		return nil, errors.Errorf("unexpect res %v", res)
	}
	lvs := make([]LvProps, 0, len(res.Report[0].LV))
	for i := 0; i < len(res.Report[0].LV); i++ {
		if res.Report[0].LV[i].LVName == "" {
			continue
		}
		lvs = append(lvs, LvProps{res.Report[0].LV[i].LVName, res.Report[0].LV[i].LVPath})
	}
	return lvs, nil
}

type VgProps struct {
	VgName string
	VgUuid string
}

type VgReports struct {
	Report []struct {
		VG []struct {
			VgName string `json:"vg_name"`
			VgUuid string `json:"vg_uuid"`
		} `json:"vg"`
	} `json:"report"`
}

func (d *LocalDiskDriver) findVg(partDev string) (*VgProps, error) {
	cmd := fmt.Sprintf("vgs --reportformat json -o vg_name,vg_uuid --devices %s 2>/dev/null", partDev)
	out, err := procutils.NewCommand("sh", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrap(err, "find vg lvs")
	}
	var vgReports VgReports
	err = json.Unmarshal(out, &vgReports)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal vgprops %s", out)
	}
	if len(vgReports.Report) == 1 && len(vgReports.Report[0].VG) == 1 {
		var vgProps VgProps
		vgProps.VgName = vgReports.Report[0].VG[0].VgName
		vgProps.VgUuid = vgReports.Report[0].VG[0].VgUuid
		return &vgProps, nil
	}

	return nil, nil
}
