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

package megactl

import (
	"fmt"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

func storcliIsJBODEnabled(
	getCmd func(args ...string) (string, error),
	term raid.IExecTerm,
) bool {
	cmd, err := getCmd("show", "jbod")
	if err != nil {
		log.Errorf("get storcli controller cmd: %v", err)
		return false
	}
	lines, err := term.Run(cmd)
	if err != nil {
		log.Errorf("storcliIsJBODEnabled error: %s", err)
		return false
	}
	for _, line := range lines {
		line = strings.ToLower(line)
		if strings.HasPrefix(line, "jbod") {
			data := strings.Split(line, " ")
			if strings.TrimSpace(data[len(data)-1]) == "on" {
				return true
			}
			return false
		}
	}
	return false
}

func storcliEnableJBOD(
	getCmd func(args ...string) (string, error),
	term raid.IExecTerm,
	enable bool) bool {
	val := "off"
	if enable {
		val = "on"
	}
	cmd, err := getCmd("set", fmt.Sprintf("jbod=%s", val), "force")
	if err != nil {
		log.Errorf("get storcli controller cmd: %v", err)
		return false
	}
	_, err = term.Run(cmd)
	if err != nil {
		log.Errorf("EnableJBOD %v fail: %v", enable, err)
		return false
	}
	return true
}

func storcliBuildJBOD(
	getCmd func(args ...string) (string, error),
	term raid.IExecTerm,
	devs []*baremetal.BaremetalStorage) error {
	if !storcliIsJBODEnabled(getCmd, term) {
		storcliEnableJBOD(getCmd, term, true)
		storcliEnableJBOD(getCmd, term, false)
		storcliEnableJBOD(getCmd, term, true)
	}
	if !storcliIsJBODEnabled(getCmd, term) {
		return fmt.Errorf("JBOD not supported")
	}
	cmds := []string{}
	for _, d := range devs {
		// cmd := GetCommand2(fmt.Sprintf("/c%d/e%d/s%d", adapter.storcliIndex, d.Enclosure, d.Slot))
		cmd, err := getCmd()
		if err != nil {
			return errors.Wrapf(err, "getCmd for dev %#v", d)
		}
		cmd = fmt.Sprintf("%s/e%d/s%d", cmd, d.Enclosure, d.Slot)
		cmds = append(cmds, cmd)
	}
	log.Infof("storcliBuildJBOD cmds: %v", cmds)
	_, err := term.Run(cmds...)
	if err != nil {
		return err
	}
	return nil
}

func storcliBuildNoRaid(
	getCmd func(args ...string) (string, error),
	term raid.IExecTerm,
	devs []*baremetal.BaremetalStorage) error {
	err := storcliBuildJBOD(getCmd, term, devs)
	if err == nil {
		return nil
	}
	log.Errorf("Try storcli build JBOD fail: %v", err)
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, GetSpecString(dev))
	}
	args := []string{
		"add", "vd", "each", "type=raid0",
		fmt.Sprintf("drives=%s", strings.Join(labels, ",")),
		"wt", "nora", "direct",
	}
	cmd, err := getCmd(args...)
	if err != nil {
		return errors.Wrapf(err, "build none raid")
	}
	_, err = term.Run(cmd)
	return err
}

func storcliClearJBODDisks(
	getCmd func(args ...string) (string, error),
	term raid.IExecTerm,
	devs []*MegaRaidPhyDev,
) error {
	errs := make([]error, 0)
	for _, dev := range devs {
		cmd, err := getCmd()
		if err != nil {
			return errors.Wrap(err, "get cmd error")
		}
		cmd = fmt.Sprintf("%s/e%d/s%d set good force", cmd, dev.enclosure, dev.slot)
		if _, err := term.Run(cmd); err != nil {
			err = errors.Wrapf(err, "Set PD good storcli cmd %v", cmd)
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func storcliBuildRaid(
	getCmd func(args ...string) (string, error),
	term raid.IExecTerm,
	devs []*baremetal.BaremetalStorage,
	conf *api.BaremetalDiskConfig,
	level uint,
) error {
	args := []string{}
	args = append(args, "add", "vd", fmt.Sprintf("type=r%d", level))
	args = append(args, conf2ParamsStorcliSize(conf)...)
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, GetSpecString(dev))
	}
	args = append(args, fmt.Sprintf("drives=%s", strings.Join(labels, ",")))
	if level == 10 {
		args = append(args, "PDperArray=2")
	}
	args = append(args, conf2ParamsStorcli(conf)...)
	cmd, err := getCmd(args...)
	if err != nil {
		return errors.Wrapf(err, "build raid %d", level)
	}
	if _, err := term.Run(cmd); err != nil {
		return err
	}
	return nil
}

func conf2ParamsStorcliSize(conf *api.BaremetalDiskConfig) []string {
	params := []string{}
	szStr := []string{}
	if len(conf.Size) > 0 {
		for _, sz := range conf.Size {
			szStr = append(szStr, fmt.Sprintf("%dMB", sz))
		}
		params = append(params, fmt.Sprintf("Size=%s", strings.Join(szStr, ",")))
	}
	return params
}

func conf2ParamsStorcli(conf *api.BaremetalDiskConfig) []string {
	params := []string{}
	if conf.WT != nil {
		if *conf.WT {
			params = append(params, "wt")
		} else {
			params = append(params, "wb")
		}
	}
	if conf.RA != nil {
		if *conf.RA {
			params = append(params, "ra")
		} else {
			params = append(params, "nora")
		}
	}
	if conf.Direct != nil {
		if *conf.Direct {
			params = append(params, "direct")
		} else {
			params = append(params, "cached")
		}
	}
	if conf.Cachedbadbbu != nil {
		if *conf.Cachedbadbbu {
			params = append(params, "CachedBadBBU")
		} else {
			params = append(params, "NoCachedBadBBU")
		}
	}
	if conf.Strip != nil {
		params = append(params, fmt.Sprintf("Strip=%d", *conf.Strip))
	}
	return params
}
