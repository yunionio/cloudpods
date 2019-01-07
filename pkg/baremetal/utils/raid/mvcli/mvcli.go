package mvcli

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type MarvelRaidPhyDev struct {
	slot    int
	adapter int
	model   string
	rotate  tristate.TriState
	sn      string
	size    int
	driver  string
}

func NewMarvelRaidPhyDev(adapter int) *MarvelRaidPhyDev {
	return &MarvelRaidPhyDev{
		slot:    -1,
		adapter: adapter,
		rotate:  tristate.None,
		driver:  baremetal.DISK_DRIVER_MARVELRAID,
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
			dev.rotate = tristate.True
		} else {
			dev.rotate = tristate.False
		}
	case "PD ID":
		dev.slot, _ = strconv.Atoi(val)
	case "Size":
		dat := strings.Split(val, " ")
		size, _ := strconv.Atoi(dat[0])
		dev.size = size / 1024 // MB
	case "model":
		dev.model = val
	case "Serial":
		dev.sn = val
	}
	return true
}

func (dev *MarvelRaidPhyDev) isComplete() bool {
	if dev.model == "" {
		return false
	}
	if dev.size < 0 {
		return false
	}
	if dev.slot < 0 {
		return false
	}
	if dev.rotate.IsNone() {
		return false
	}
	if dev.sn == "" {
		return false
	}
	return true
}

func (dev *MarvelRaidPhyDev) String() string {
	return fmt.Sprintf("%d", dev.slot)
}

type MarvelRaidAdaptor struct {
	index int
	raid  *MarvelRaid
	devs  []*baremetal.BaremetalStorage
}

func NewMarvelRaidAdaptor(index int, raid *MarvelRaid) *MarvelRaidAdaptor {
	return &MarvelRaidAdaptor{
		index: index,
		raid:  raid,
	}
}

func (adapter *MarvelRaidAdaptor) ParsePhyDevs() bool {
	cmd := adapter.raid.GetCommand("info", "-o", "pd")
	ret, err := adapter.raid.term.Run(cmd)
	if err != nil {
		log.Errorf("get physical device: %v", err)
		return false
	}
	return adapter.parsePhyDevs(ret)
}

func (adapter *MarvelRaidAdaptor) parsePhyDevs(lines []string) bool {
	phyDev := NewMarvelRaidPhyDev(adapter.index)
	for _, line := range lines {
		if phyDev.parseLine(line) && phyDev.isComplete() {
			adapter.devs.append(phyDev)
			phyDev = NewMarvelRaidPhyDev(adapter.index)
		}
	}
}

func (adapter) GetPhyDevs() []*MarvelRaidPhyDev {
	return adapter.devs
}

type MarvelRaid struct {
	term     *ssh.Client
	adapters []*MarvelRaidAdaptor
	lv       []interface{}
}

func GetCommand(args ...string) string {
	bin := "/opt/mvcli/mvcli"
	return raid.GetCommand(bin, args...)
}

func (r *MarvelRaid) ParsePhyDevs() bool {
	cmd := GetCommand("info", "-o", "hba")
	ret, err := r.term.Run(cmd)
	if err != nil {
		log.Errorf("Remote get info error: %v", err)
		return false
	}
	err = r.parseAdapters(ret)
	if err != nil {
		log.Errorf("parse adapt error: %v", err)
	}
	if len(r.adapters) > 0 {
		return true
	}
	log.Errorf("Empty adapters")
	return false
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
