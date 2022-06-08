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

package adaptec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

func init() {
	raid.RegisterDriver(api.DISK_DRIVER_ADAPTECRAID, NewAdaptecRaid)
}

const (
	ControllerModeRaidExposeRaw = "RAID (Expose RAW)"
	ControllerModeRaidHideRaw   = "RAID (Hide RAW)"
	ControllerModeMixed         = "Mixed"
)

type AdaptecRaid struct {
	term     raid.IExecTerm
	adapters []*AdaptecRaidAdaptor
}

func NewAdaptecRaid(term raid.IExecTerm) raid.IRaidDriver {
	return &AdaptecRaid{
		term:     term,
		adapters: make([]*AdaptecRaidAdaptor, 0),
	}
}

func GetCommand(args ...string) string {
	bin := "/opt/adaptec/arcconf"
	return raid.GetCommand(bin, args...)
}

func (r *AdaptecRaid) GetName() string {
	return baremetal.DISK_DRIVER_ADAPTECRAID
}

func (r *AdaptecRaid) ParsePhyDevs() error {
	cmd := GetCommand("LIST")
	ret, err := r.term.Run(cmd)
	if err != nil {
		return errors.Wrap(err, "list controllers")
	}
	if err := r.parsePhyDevs(ret); err != nil {
		return errors.Wrap(err, "parse physical device")
	}
	if len(r.adapters) == 0 {
		return errors.Errorf("Not found adaptec raid controller")
	}
	return nil
}

func (r *AdaptecRaid) CleanRaid() error {
	for _, ada := range r.adapters {
		ada.removeJBODDisks()
		ada.RemoveLogicVolumes()
	}
	return nil
}

func (r *AdaptecRaid) GetAdapters() []raid.IRaidAdapter {
	ret := make([]raid.IRaidAdapter, 0)
	for _, a := range r.adapters {
		ret = append(ret, a)
	}
	return ret
}

func (r *AdaptecRaid) PreBuildRaid(confs []*api.BaremetalDiskConfig, adapterIdx int) error {
	return nil
}

var (
	adaptorIDPatter = regexp.MustCompile(`Controller (?P<idx>[0-9]+):`)
)

func getAdaptorIndex(line string) int {
	adapStr := regutils2.GetParams(adaptorIDPatter, line)["idx"]
	if adapStr == "" {
		return -1
	}
	adapInt, err := strconv.Atoi(adapStr)
	if err != nil {
		log.Errorf("Parse adapator string %q id error: %v", line, err)
		return -1
	}
	return adapInt
}

func (r *AdaptecRaid) parsePhyDevs(lines []string) error {
	for _, line := range lines {
		index := getAdaptorIndex(line)
		if index == -1 {
			continue
		}
		ada, err := NewAdaptecRaidAdaptor(index, r)
		if err != nil {
			return errors.Wrapf(err, "New raid adaptor %d", index)
		}
		r.adapters = append(r.adapters, ada)
	}
	return nil
}

type AdaptecRaidAdaptor struct {
	*adaptorInfo

	index int
	raid  *AdaptecRaid
	devs  []*AdaptecRaidPhyDev
}

var (
	_ raid.IRaidAdapter = new(AdaptecRaidAdaptor)
)

func NewAdaptecRaidAdaptor(index int, raid *AdaptecRaid) (*AdaptecRaidAdaptor, error) {
	adaptor := AdaptecRaidAdaptor{
		index: index,
		raid:  raid,
	}
	if err := adaptor.fillInfo(); err != nil {
		return nil, errors.Wrapf(err, "fill adaptor %d info", adaptor.index)
	}
	if err := adaptor.fillDevices(); err != nil {
		return nil, errors.Wrap(err, "fill physical devices")
	}
	/*
	 * if !adaptor.isRaidHideRawMode() {
	 * 	if err := adaptor.setControllerModeRaidHideRaw(); err != nil {
	 * 		return nil, errors.Wrap(err, "set controller mode to raid hide raw")
	 * 	}
	 * }
	 */
	return &adaptor, nil
}

func (ada *AdaptecRaidAdaptor) GetIndex() int {
	return ada.index
}

func (ada *AdaptecRaidAdaptor) PreBuildRaid(confs []*api.BaremetalDiskConfig) error {
	if !ada.isRaidHideRawMode() {
		if err := ada.setControllerModeRaidHideRaw(); err != nil {
			return errors.Wrapf(err, "set controller mode to raid hide raw mode")
		}
	}
	if err := ada.removeJBODDisks(); err != nil {
		log.Warningf("remove all JBOD disks: %v", err)
	}
	// uninitialize all device to ready state
	if err := ada.uninitializeAllDevice(); err != nil {
		// return errors.Wrap(err, "uninitialize all devices")
		log.Warningf("uninitializeAllDevice error: %v", err)
	}
	return nil
}

func (r *AdaptecRaidAdaptor) PostBuildRaid() error {
	return nil
}

func (ada *AdaptecRaidAdaptor) getInitializeCmd(args ...string) string {
	newArgs := []string{"TASK", "START", fmt.Sprintf("%d", ada.GetIndex()), "DEVICE"}
	newArgs = append(newArgs, args...)
	newArgs = append(newArgs, "INITIALIZE", "noprompt")
	return GetCommand(newArgs...)
}

func (ada *AdaptecRaidAdaptor) initializeDevice(channel string, id string) error {
	cmd := ada.getInitializeCmd(channel, id)
	_, err := ada.remoteRun(fmt.Sprintf("initialize device %s:%s", channel, id), cmd)
	if err != nil {
		return err
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) initializeAllDevice() error {
	cmd := ada.getInitializeCmd("ALL")
	_, err := ada.remoteRun("initialize all device", cmd)
	if err != nil {
		return err
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) uninitializeDevice(channel string, id string) error {
	cmd := ada.getUninitializeCmd(channel, id)
	_, err := ada.remoteRun(fmt.Sprintf("uninitialize device %s:%s", channel, id), cmd)
	if err != nil {
		return err
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) getUninitializeCmd(args ...string) string {
	newArgs := []string{"TASK", "START", fmt.Sprintf("%d", ada.GetIndex()), "DEVICE"}
	newArgs = append(newArgs, args...)
	newArgs = append(newArgs, "UNINITIALIZE", "noprompt")
	return GetCommand(newArgs...)
}

func (ada *AdaptecRaidAdaptor) uninitializeAllDevice() error {
	cmd := ada.getUninitializeCmd("ALL")
	_, err := ada.remoteRun("uninitialize all devices", cmd)
	if err != nil {
		return err
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) getTerm() raid.IExecTerm {
	return ada.raid.term
}

func (ada *AdaptecRaidAdaptor) remoteRun(hint string, cmd string) ([]string, error) {
	out, err := ada.getTerm().Run(cmd)
	if err != nil {
		return out, errors.Wrapf(err, "%q, out: %v", hint, out)
	}
	log.Debugf("remote run cmd %s %q successfully: %v", hint, cmd, out)
	return out, nil
}

func (ada *AdaptecRaidAdaptor) fillInfo() error {
	cmd := GetCommand("GETCONFIG", fmt.Sprintf("%d", ada.index), "AD")
	ret, err := ada.remoteRun("get AD config", cmd)
	if err != nil {
		return err
	}
	info, err := getAdaptorInfo(ret)
	if err != nil {
		return errors.Wrap(err, "get adaptor info")
	}
	ada.adaptorInfo = info
	return nil
}

func (ada *AdaptecRaidAdaptor) fillDevices() error {
	cmd := GetCommand("GETCONFIG", fmt.Sprintf("%d", ada.index), "PD")
	ret, err := ada.remoteRun("get PD config", cmd)
	if err != nil {
		return err
	}
	devs := getPhyDevices(ada.GetIndex(), ret)
	ada.devs = append(ada.devs, devs...)
	return nil
}

func getPhyDevices(adapter int, lines []string) []*AdaptecRaidPhyDev {
	dev := newPhyDev(adapter)
	devs := make([]*AdaptecRaidPhyDev, 0)
	for _, line := range lines {
		if dev.parseLine(line) && dev.isComplete() {
			tmpDev := dev
			devs = append(devs, tmpDev)
			dev = newPhyDev(adapter)
		}
	}
	return devs
}

func (ada *AdaptecRaidAdaptor) getChannelId(storage *baremetal.BaremetalStorage) (string, string, error) {
	if storage.Addr == "" {
		return "", "", errors.Errorf("storage %#v addr is empty", storage)
	}
	parts := strings.Split(storage.Addr, ":")
	if len(parts) != 2 {
		return "", "", errors.Errorf("invalid storage %#v addr %q", storage, storage.Addr)
	}
	return parts[0], parts[1], nil
}

func (ada *AdaptecRaidAdaptor) getCreateCmd(args ...string) string {
	newArgs := []string{"CREATE", fmt.Sprintf("%d", ada.GetIndex())}
	newArgs = append(newArgs, args...)
	return GetCommand(newArgs...)
}

// setControllerMode change adapter controller's mode
//   Controller Modes  : 0  - RAID: Expose RAW
//                     : 1  - Auto Volume Mode
//                     : 2  - HBA Mode
//                     : 3  - RAID: Hide RAW
//                     : 4  - Simple Volume Mode
//                     : 5  - Mixed
func (ada *AdaptecRaidAdaptor) setControllerMode(mode int) error {
	cmd := GetCommand("SETCONTROLLERMODE", fmt.Sprintf("%d", ada.GetIndex()), fmt.Sprintf("%d", mode), "noprompt")
	if _, err := ada.remoteRun(fmt.Sprintf("set controller mode to %d", mode), cmd); err != nil {
		return err
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) setControllerModeRaidExposeRaw() error {
	if err := ada.setControllerMode(0); err != nil {
		return errors.Wrap(err, "mode raid expose raw")
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) setControllerModeRaidHideRaw() error {
	if err := ada.setControllerMode(3); err != nil {
		return errors.Wrap(err, "mode raid hide raw")
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) setControllerModeMixed() error {
	if err := ada.setControllerMode(5); err != nil {
		return errors.Wrap(err, "mode mixed")
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) buildJBOD(dev *baremetal.BaremetalStorage) error {
	channel, id, err := ada.getChannelId(dev)
	if err != nil {
		return errors.Wrap(err, "get channel and id")
	}
	cmd := ada.getCreateCmd("JBOD", channel, id, "noprompt")
	if out, err := ada.remoteRun("create JBOD", cmd); err != nil {
		return errors.Wrapf(err, "run cmd %q, out: %q", cmd, out)
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) buildNonRaid(dev *baremetal.BaremetalStorage) error {
	// set to raid expose raw mode or mixed mode
	if !ada.isRaidExposeRawMode() && !ada.isMixedMode() {
		err := func() error {
			errs := []error{}
			if err := ada.setControllerModeRaidExposeRaw(); err != nil {
				errs = append(errs, err)
			} else {
				return nil
			}
			if err := ada.setControllerModeMixed(); err != nil {
				errs = append(errs, err)
			} else {
				return nil
			}
			return errors.NewAggregate(errs)
		}()
		if err != nil {
			return errors.Wrap(err, "set raid to expose raw or mixed mode")
		}
	}
	// try build JBOD firstly
	if err := ada.buildJBOD(dev); err != nil {
		log.Warningf("try build JBOD error: %v", err)
	} else {
		return nil
	}
	// just uninitialize device when build JBOD fail
	channel, id, err := ada.getChannelId(dev)
	if err != nil {
		return err
	}
	if err := ada.uninitializeDevice(channel, id); err != nil {
		return errors.Wrapf(err, "uninitialize device %v when build jbod fail", dev)
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) BuildNoneRaid(devs []*baremetal.BaremetalStorage) error {
	for _, dev := range devs {
		if err := ada.buildNonRaid(dev); err != nil {
			return errors.Wrap(err, "build nonraid")
		}
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) getBuildRaidCmd(level int, devs []*baremetal.BaremetalStorage) (string, error) {
	if len(devs) == 0 {
		return "", errors.Errorf("devices is empty")
	}
	args := []string{"LOGICALDRIVE", "MAX", fmt.Sprintf("%d", level)}
	chIds := []string{}
	for _, dev := range devs {
		ch, id, err := ada.getChannelId(dev)
		if err != nil {
			return "", errors.Wrapf(err, "get device %v channel id", dev)
		}
		chIds = append(chIds, ch, id)
	}
	args = append(args, chIds...)
	cmd := ada.getCreateCmd(args...)
	return cmd, nil
}

func (ada *AdaptecRaidAdaptor) buildRaid(level int, devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	// initialize each devs
	for _, dev := range devs {
		channel, id, err := ada.getChannelId(dev)
		if err != nil {
			return errors.Wrapf(err, "get device %v channel id", dev)
		}
		if err := ada.initializeDevice(channel, id); err != nil {
			// return errors.Wrapf(err, "initialize device")
			log.Warningf("initialize device error: %v", err)
		}
	}

	// TODO: support config to build raid params
	cmd, err := ada.getBuildRaidCmd(level, devs)
	if err != nil {
		return err
	}
	_, err = ada.remoteRun(fmt.Sprintf("build raid %d", level), cmd)
	return err
}

func (ada *AdaptecRaidAdaptor) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return ada.buildRaid(0, devs, conf)
}

func (ada *AdaptecRaidAdaptor) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return ada.buildRaid(1, devs, conf)
}

func (ada *AdaptecRaidAdaptor) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return ada.buildRaid(5, devs, conf)
}

func (ada *AdaptecRaidAdaptor) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return ada.buildRaid(10, devs, conf)
}

func (ada *AdaptecRaidAdaptor) GetDevices() []*baremetal.BaremetalStorage {
	ret := []*baremetal.BaremetalStorage{}
	for idx, dev := range ada.devs {
		ret = append(ret, dev.ToBaremetalStorage(idx))
	}
	return ret
}

func (ada *AdaptecRaidAdaptor) GetLogicVolumes() ([]*raid.RaidLogicalVolume, error) {
	cmd := GetCommand("GETCONFIG", fmt.Sprintf("%d", ada.index), "LD")
	ret, err := ada.remoteRun("get logic volumes", cmd)
	if err != nil {
		return nil, fmt.Errorf("Get logic volumes error: %v", err)
	}
	return getLogicalVolumes(ada.index, ret)
}

func getLogicalVolumes(adapter int, lines []string) ([]*raid.RaidLogicalVolume, error) {
	lvs := make([]*raid.RaidLogicalVolume, 0)
	for _, line := range lines {
		m := regutils2.SubGroupMatch(`Logical Device number\s+(?P<idx>\d+)`, line)
		if len(m) > 0 {
			idxStr := m["idx"]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, errors.Errorf("%s index str is not digit: %v", idxStr, err)
			}
			lvs = append(lvs, &raid.RaidLogicalVolume{
				Index:   idx,
				Adapter: adapter,
			})
		}
	}
	return lvs, nil
}

func (ada *AdaptecRaidAdaptor) removeJBODDisks() error {
	cmd := GetCommand("DELETE", fmt.Sprintf("%d", ada.index), "JBOD", "ALL", "noprompt")
	out, err := ada.remoteRun("delete JBOD disks", cmd)
	if err != nil {
		return errors.Wrapf(err, "delete all JBOD output: %q", out)
	}
	return nil
}

func (ada *AdaptecRaidAdaptor) RemoveLogicVolumes() error {
	lvs, err := ada.GetLogicVolumes()
	if err != nil {
		return errors.Wrap(err, "get logic volumes")
	}
	if len(lvs) == 0 {
		return nil
	}
	cmd := GetCommand("DELETE", fmt.Sprintf("%d", ada.index), "LOGICALDRIVE", "ALL", "noprompt")
	out, err := ada.remoteRun("delete logical volumes", cmd)
	if err != nil {
		return errors.Wrapf(err, "delete all logicaldrive output: %q", out)
	}
	return nil
}

type adaptorInfo struct {
	status string
	mode   string
	name   string
	sn     string
	wwn    string
	slot   string
}

func (ada *adaptorInfo) key() string {
	return ada.name + ada.sn
}

func (ada *adaptorInfo) isRaidExposeRawMode() bool {
	return ada.mode == ControllerModeRaidExposeRaw
}

func (ada *adaptorInfo) isRaidHideRawMode() bool {
	return ada.mode == ControllerModeRaidHideRaw
}

func (ada *adaptorInfo) isMixedMode() bool {
	return ada.mode == ControllerModeMixed
}

func getAdaptorInfo(lines []string) (*adaptorInfo, error) {
	ada := new(adaptorInfo)
	for _, l := range lines {
		key, val := stringutils.SplitKeyValue(l)
		if len(key) == 0 {
			continue
		}
		switch key {
		case "Controller Status":
			ada.status = val
		case "Controller Mode":
			ada.mode = val
		case "Controller Model":
			ada.name = val
		case "Controller Serial Number":
			ada.sn = val
		case "Controller World Wide Name":
			ada.wwn = val
		case "Physical Slot":
			ada.slot = val
		}
	}
	if len(ada.key()) == 0 {
		return nil, errors.Errorf("Not found SN and model name")
	}
	return ada, nil
}

type AdaptecRaidPhyDev struct {
	*raid.RaidBasePhyDev
	// channelId and deviceId is parsed by Reported Channel,Device(T:L)
	// e.g.: Reported Channel,Device(T:L)       : 0,6(6:0)
	channelId string
	deviceId  string
}

func newPhyDev(adapter int) *AdaptecRaidPhyDev {
	dev := &AdaptecRaidPhyDev{
		RaidBasePhyDev: raid.NewRaidBasePhyDev(api.DISK_DRIVER_ADAPTECRAID),
	}
	dev.Adapter = adapter
	return dev
}

func (dev *AdaptecRaidPhyDev) isComplete() bool {
	if !dev.RaidBasePhyDev.IsComplete() {
		return false
	}
	if dev.channelId == "" || dev.deviceId == "" {
		return false
	}
	return true
}

func (dev *AdaptecRaidPhyDev) ToBaremetalStorage(index int) *baremetal.BaremetalStorage {
	s := dev.RaidBasePhyDev.ToBaremetalStorage(index)
	s.Addr = fmt.Sprintf("%s:%s", dev.channelId, dev.deviceId)

	return s
}

var (
	channelDeviceRegexp = regexp.MustCompile(`Reported Channel,Device\(T:L\).*(?P<channel>\d+),(?P<device>\d+)\(\d+:\d+\)`)
)

func (dev *AdaptecRaidPhyDev) parseLine(line string) bool {
	chanDevMatch := regutils2.GetParams(channelDeviceRegexp, line)
	if len(chanDevMatch) != 0 {
		channelId := chanDevMatch["channel"]
		deviceId := chanDevMatch["device"]
		dev.channelId = channelId
		dev.deviceId = deviceId
		return true
	}
	key, val := stringutils.SplitKeyValue(line)
	if key == "" {
		return false
	}
	switch key {
	case "Total Size":
		dat := strings.Split(val, " ")
		szStr, unitStr := dat[0], dat[1]
		var sz int64
		szInt, err := strconv.Atoi(szStr)
		if err != nil {
			log.Errorf("Parse size string %s: %v", szStr, err)
			return false
		}
		switch unitStr {
		case "GB":
			sz = int64(szInt * 1000 * 1000 * 100)
		case "TB":
			sz = int64(szInt * 1000 * 1000 * 1000 * 1000)
		case "MB":
			sz = int64(szInt * 1000 * 1000)
		default:
			log.Errorf("Unsupported unit: %s", unitStr)
			return false
		}
		dev.Size = sz / 1024 / 1024
	case "Model":
		dev.Model = strings.Join(regexp.MustCompile(`\s+`).Split(val, -1), " ")
	case "State":
		dev.Status = val
	case "SSD":
		if val == "No" {
			dev.Rotate = tristate.True
		} else {
			dev.Rotate = tristate.False
		}
	default:
		return false
	}
	return true
}
