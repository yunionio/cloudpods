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

package fsutils

import (
	"encoding/json"
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

func VgActive(vgname string) error {
	out, err := procutils.NewCommand("vgchange", "-ay", vgname).Output()
	if err != nil {
		return errors.Wrapf(err, "vgchange -ay %s %s", vgname, out)
	}
	return nil
}

func (d *SFsutilDriver) ExtendLv(lvPath string) error {
	out, err := d.Exec("lvextend", "-l", "+100%FREE", lvPath)
	if err != nil {
		return errors.Wrapf(err, "extend lv %s failed %s", lvPath, out)
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

func (d *SFsutilDriver) GetVgLvs(vg string) ([]LvProps, error) {
	cmd := fmt.Sprintf("lvs --reportformat json -o lv_name,lv_path %s 2>/dev/null", vg)
	out, err := d.Exec("sh", "-c", cmd)
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

func FindVg(partDev string) (*VgProps, error) {
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
