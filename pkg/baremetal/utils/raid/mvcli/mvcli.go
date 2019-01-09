package mvcli

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type MarvelRaidPhyDev struct {
	*raid.RaidBasePhyDev
	slot int
	sn   string
}

func NewMarvelRaidPhyDev(adapter int) *MarvelRaidPhyDev {
	b := raid.NewRaidBasePhyDev(baremetal.DISK_DRIVER_MARVELRAID)
	b.Adapter = adapter
	return &MarvelRaidPhyDev{
		RaidBasePhyDev: b,
		slot:           -1,
	}
}

func (dev *MarvelRaidPhyDev) parseLine(line string) bool {
	key, val := stringutils.SplitKeyValue(line)
	if key == "" {
		return false
	}
	switch key {
	case "SSD Type":
		if strings.HasSuffix(val, "SSD") {
			dev.Rotate = tristate.True
		} else {
			dev.Rotate = tristate.False
		}
	case "PD ID":
		dev.slot, _ = strconv.Atoi(val)
	case "Size":
		dat := strings.Split(val, " ")
		size, _ := strconv.Atoi(dat[0])
		dev.Size = int64(size / 1024) // MB
	case "model":
		dev.Model = val
	case "Serial":
		dev.sn = val
	default:
		return false
	}
	return true
}

func (dev *MarvelRaidPhyDev) isComplete() bool {
	if !dev.RaidBasePhyDev.IsComplete() {
		return false
	}
	if dev.Size < 0 {
		return false
	}
	if dev.slot < 0 {
		return false
	}
	if dev.sn == "" {
		return false
	}
	return true
}

func (dev *MarvelRaidPhyDev) ToBaremetalStorage() *baremetal.BaremetalStorage {
	s := dev.RaidBasePhyDev.ToBaremetalStorage()
	s.Slot = dev.slot
	return s
}

func GetSpecString(dev *baremetal.BaremetalStorage) string {
	return fmt.Sprintf("%d", dev.Slot)
}

type MarvelRaidAdaptor struct {
	index int
	raid  *MarvelRaid
	devs  []*MarvelRaidPhyDev
}

func NewMarvelRaidAdaptor(index int, raid *MarvelRaid) *MarvelRaidAdaptor {
	return &MarvelRaidAdaptor{
		index: index,
		raid:  raid,
		devs:  make([]*MarvelRaidPhyDev, 0),
	}
}

func (adapter *MarvelRaidAdaptor) GetIndex() int {
	return adapter.index
}

func (adapter *MarvelRaidAdaptor) ParsePhyDevs() error {
	cmd := GetCommand("info", "-o", "pd")
	ret, err := adapter.raid.term.Run(cmd)
	if err != nil {
		return fmt.Errorf("get physical device: %v", err)
	}
	return adapter.parsePhyDevs(ret)
}

func (adapter *MarvelRaidAdaptor) parsePhyDevs(lines []string) error {
	phyDev := NewMarvelRaidPhyDev(adapter.index)
	for _, line := range lines {
		if phyDev.parseLine(line) && phyDev.isComplete() {
			adapter.devs = append(adapter.devs, phyDev)
			phyDev = NewMarvelRaidPhyDev(adapter.index)
		}
	}
	return nil
}

func (adapter *MarvelRaidAdaptor) GetDevices() []*baremetal.BaremetalStorage {
	ret := []*baremetal.BaremetalStorage{}
	for _, dev := range adapter.devs {
		ret = append(ret, dev.ToBaremetalStorage())
	}
	return ret
}

func (adapter *MarvelRaidAdaptor) GetLogicVolumes() ([]int, error) {
	cmd := GetCommand("info", "-o", "vd")
	ret, err := adapter.raid.term.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("getLogicVolumes: %v", err)
	}
	return adapter.parseLogicVolumes(ret)
}

func (adapter *MarvelRaidAdaptor) parseLogicVolumes(lines []string) ([]int, error) {
	lvIdx := []int{}
	usedDevs := []int{}
	for _, line := range lines {
		key, val := stringutils.SplitKeyValue(line)
		if key != "" {
			if key == "id" {
				idx, err := strconv.Atoi(val)
				if err != nil {
					return nil, err
				}
				lvIdx = append(lvIdx, idx)
			} else if key == "PD RAID setup" {
				for _, d := range strings.Split(val, " ") {
					idx, err := strconv.Atoi(d)
					if err != nil {
						return nil, err
					}
					usedDevs = append(usedDevs, idx)
				}
			}
		}
	}
	if len(adapter.devs) < len(usedDevs) {
		return nil, fmt.Errorf("adapter %d current %d devs < usedDevs %d", adapter.index, len(adapter.devs), len(usedDevs))
	}
	return lvIdx, nil
}

func (adapter *MarvelRaidAdaptor) RemoveLogicVolumes() error {
	lvs, err := adapter.GetLogicVolumes()
	if err != nil {
		return fmt.Errorf("Failed to get logic volumes: %v", err)
	}
	for i := len(lvs) - 1; i >= 0; i-- {
		if err := adapter.removeLogicVolume(i); err != nil {
			return fmt.Errorf("Remove %d logical volume: %v", i, err)
		}
	}
	return nil
}

func (adapter *MarvelRaidAdaptor) removeLogicVolume(idx int) error {
	cmd := GetCommand("delete", "-o", "vd", "-i", fmt.Sprintf("%d", idx), "-f", "--waiveconfirmation")
	_, err := adapter.raid.term.Run(cmd)
	return err
}

func (adapter *MarvelRaidAdaptor) PreBuildRaid(confs []*baremetal.BaremetalDiskConfig) error {
	return nil
}

func (adapter *MarvelRaidAdaptor) buildRaid(level string, devs []*baremetal.BaremetalStorage, _ *baremetal.BaremetalDiskConfig) error {
	pds := []string{}
	for _, dev := range devs {
		pds = append(pds, fmt.Sprintf("%s", GetSpecString(dev)))
	}
	args := []string{"create", "-o", "vd", "-d", strings.Join(pds, ","), level, "--waiveconfirmation"}
	cmd := GetCommand(args...)
	_, err := adapter.raid.term.Run(cmd)
	return err
}

func (adapter *MarvelRaidAdaptor) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter.buildRaid("-r0", devs, conf)
}

func (adapter *MarvelRaidAdaptor) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter.buildRaid("-r1", devs, conf)
}

func (adapter *MarvelRaidAdaptor) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	//return adapter.buildRaid("-r5", devs, conf)
	return fmt.Errorf("BuildRaid5 not impl")
}

func (adapter *MarvelRaidAdaptor) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter.buildRaid("-r10", devs, conf)
}

func (adapter *MarvelRaidAdaptor) BuildNoneRaid(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return fmt.Errorf("BuildNoneRaid not impl")
}

type MarvelRaid struct {
	term     *ssh.Client
	adapters []*MarvelRaidAdaptor
}

func NewMarvelRaid(term *ssh.Client) raid.IRaidDriver {
	return &MarvelRaid{
		term:     term,
		adapters: make([]*MarvelRaidAdaptor, 0),
	}
}

func (r *MarvelRaid) GetName() string {
	return baremetal.DISK_DRIVER_MARVELRAID
}

func GetCommand(args ...string) string {
	bin := "/opt/mvcli/mvcli"
	return raid.GetCommand(bin, args...)
}

func (r *MarvelRaid) ParsePhyDevs() error {
	cmd := GetCommand("info", "-o", "hba")
	ret, err := r.term.Run(cmd)
	if err != nil {
		return fmt.Errorf("Remote get info error: %v", err)
	}
	err = r.parseAdapters(ret)
	if err != nil {
		return fmt.Errorf("parse adapt error: %v", err)
	}
	if len(r.adapters) > 0 {
		return nil
	}
	return fmt.Errorf("Empty adapters")
}

func (r *MarvelRaid) parseAdapters(lines []string) error {
	for _, line := range lines {
		k, v := stringutils.SplitKeyValue(line)
		if k == "Adapter ID" {
			vi, err := strconv.Atoi(v)
			if err != nil {
				return err
			}
			adapter := NewMarvelRaidAdaptor(vi, r)
			r.adapters = append(r.adapters, adapter)
		}
	}
	for _, adp := range r.adapters {
		adp.ParsePhyDevs()
	}
	return nil
}

func (r *MarvelRaid) PreBuildRaid(_ []*baremetal.BaremetalDiskConfig, _ int) error {
	return nil
}

func (r *MarvelRaid) GetAdapters() []raid.IRaidAdapter {
	ret := make([]raid.IRaidAdapter, 0)
	for _, a := range r.adapters {
		ret = append(ret, a)
	}
	return ret
}

func (r *MarvelRaid) CleanRaid() error {
	// pass
	return nil
}

func init() {
	raid.RegisterDriver(baremetal.DISK_DRIVER_MARVELRAID, NewMarvelRaid)
}
