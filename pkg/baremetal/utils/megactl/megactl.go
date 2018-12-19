package megactl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

var (
	sizePattern   = regexp.MustCompile(`(?P<sector>0x[0-9a-fA-F]+)`)
	adapterPatter = regexp.MustCompile(`^Adapter #(?P<idx>[0-9]+)`)
)

type MegaRaidPhyDev struct {
	enclosure    int
	slot         int
	minStripSize int
	maxStripSize int
	model        string
	rotate       tristate.TriState
	adapter      int
	status       string
	driver       string
	sector       int
	block        int
}

func NewMegaRaidPhyDev() *MegaRaidPhyDev {
	return &MegaRaidPhyDev{
		enclosure:    -1,
		slot:         -1,
		minStripSize: -1,
		maxStripSize: -1,
		rotate:       tristate.None,
		driver:       baremetal.DISK_DRIVER_MEGARAID,
		sector:       -1,
		block:        512,
	}
}

func (dev *MegaRaidPhyDev) GetSize() int {
	return dev.sector * dev.block / 1024 / 1024 // MB
}

func (dev *MegaRaidPhyDev) parseLine(line string) bool {
	key, val := stringutils.SplitKeyValue(line)
	if key == "" {
		return false
	}
	switch key {
	case "Media Type":
		if val == "Hard Disk Device" {
			dev.rotate = tristate.True
		} else {
			dev.rotate = tristate.False
		}
	case "Enclosure Device ID":
		enclosure, err := strconv.Atoi(val)
		if err == nil {
			dev.enclosure = enclosure
		}
	case "Slot Number":
		dev.slot, _ = strconv.Atoi(val)
	case "Coerced Size":
		matches := sizePattern.FindStringSubmatch(val)
		if len(matches) != 0 {
			paramsMap := make(map[string]string)
			for i, name := range sizePattern.SubexpNames() {
				if i > 0 && i <= len(matches) {
					paramsMap[name] = matches[i]
				}
			}
			sizeStr := paramsMap["sector"]
			sector, err := strconv.ParseInt(sizeStr, 16, 32)
			if err != nil {
				dev.sector = 0
			}
			dev.sector = int(sector)
		} else {
			dev.sector = 0
		}
	case "Inquiry Data":
		dev.model = strings.Join(regexp.MustCompile(`\s+`).Split(val, -1), " ")
	case "Firmware state":
		if val == "JBOD" {
			dev.status = "jbod"
		} else if strings.Contains(strings.ToLower(val), "online") {
			dev.status = "online"
		} else {
			dev.status = "offline"
		}
	case "Logical Sector Size":
		block, err := strconv.Atoi(val)
		if err != nil {
			log.Errorf("parse logical sector size error: %v", err)
			dev.block = 512
		} else {
			dev.block = block
		}
	}
	return true
}

func (dev *MegaRaidPhyDev) parseStripSize(lines []string) bool {
	size2Int := func(sizeStr string) int {
		// TODO
		return -1
	}
	for _, line := range lines {
		if strings.Contains(line, "Min") {
			dev.minStripSize = size2Int(strings.Split(line, ": ")[1])
		}
		if strings.Contains(line, "Max") {
			dev.maxStripSize = size2Int(strings.Split(line, ": ")[1])
		}
	}
	return true
}

func (dev *MegaRaidPhyDev) isComplete() bool {
	if dev.model == "" {
		return false
	}
	if dev.sector < 0 {
		return false
	}
	if dev.block < 0 {
		return false
	}
	if dev.slot < 0 {
		return false
	}
	if dev.rotate.IsNone() {
		return false
	}
	if dev.status == "" {
		return false
	}
	return true
}

func (dev *MegaRaidPhyDev) isJBOD() bool {
	return dev.status == "jbod"
}

func (dev *MegaRaidPhyDev) String() string {
	if dev.enclosure < 0 {
		return fmt.Sprintf(":%d", dev.slot)
	}
	return fmt.Sprintf("%d:%d", dev.enclosure, dev.slot)
}

type MegaRaidAdaptor struct {
	index int
	raid  *MegaRaid
	devs  []*MegaRaidPhyDev
	lvs   []interface{}
}

func NewMegaRaidAdaptor(index int, raid *MegaRaid) *MegaRaidAdaptor {
	return &MegaRaidAdaptor{
		index: index,
		raid:  raid,
	}
}

func (adapter *MegaRaidAdaptor) AddPhyDev(dev *MegaRaidPhyDev) {
	dev.adapter = adapter.index
	adapter.devs = append(adapter.devs, dev)
}

func (adapter *MegaRaidAdaptor) GetPhyDevs() []*MegaRaidPhyDev {
	return adapter.devs
}

func (adapter *MegaRaidAdaptor) getLogicVolumes() []int {
	cmd := GetCommand("-LDInfo", "-Lall", fmt.Sprintf("-a%d", adapter.index))
	ret, err := adapter.remoteRun(cmd)
	if err != nil {
		return nil
	}
	return adapter.parseLogicVolumes(ret)
}

func (adapter *MegaRaidAdaptor) parseLogicVolumes(lines []string) []int {
	lvIdx := []int{}
	for _, line := range lines {
		key, val := stringutils.SplitKeyValue(line)
		if key != "" && key == "Virtual Drive" {
			idx, _ := strconv.Atoi(strings.Split(val, " ")[0])
			lvIdx = append(lvIdx, idx)
		}
	}
	return lvIdx
}

func (adapter *MegaRaidAdaptor) BuildRaid(confs interface{}) bool {

}

func (adapter *MegaRaidAdaptor) getTerm() *ssh.Client {
	return adapter.raid.term
}

func (adapter *MegaRaidAdaptor) remoteRun(cmds ...string) ([]string, error) {
	return adapter.getTerm().Run(cmds...)
}

func (adapter *MegaRaidAdaptor) megacliIsJBODEnabled() bool {
	cmd := GetCommand("-AdpGetProp", "-EnableJBOD", fmt.Sprintf("-a%d", adapter.index))
	pref := fmt.Sprintf("Adapter %d: JBOD: ", adapter.index)
	lines, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("megacliIsJBODEnabled error: %v", err)
		return false
	}
	for _, line := range lines {
		if strings.HasPrefix(line, pref) {
			val := strings.ToLower(strings.TrimSpace(line[len(pref):]))
			if val == "disabled" {
				return false
			}
			return true
		}
	}
	return false
}

func (adapter *MegaRaidAdaptor) megacliEnableJBOD(enable bool) bool {
	var val string = "0"
	if enable {
		val = "1"
	}
	cmd := GetCommand("-AdpSetProp", "-EnableJBOD", fmt.Sprintf("-%s", val), fmt.Sprintf("-a%d", adapter.index))
	_, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("enable jbod %v fail: %v", enable, err)
		return false
	}
	return true
}

func (adapter *MegaRaidAdaptor) megacliClearJBODDisks() error {
	devIds := []string{}
	for _, dev := range adapter.devs {
		devIds = append(devIds, dev.String())
	}
	cmd := GetCommand("-PDMakeGood", fmt.Sprintf("-PhysDrv[%s]", strings.Join(devIds, ",")), "-Force", fmt.Sprintf("-a%d", adapter.index))
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) clearJBODDisks() {
	adapter.megacliEnableJBOD(true)
	adapter.megacliEnableJBOD(false)
	adapter.megacliEnableJBOD(true)
	adapter.megacliEnableJBOD(false)
}

type MegaRaid struct {
	term       *ssh.Client
	adapters   []*MegaRaidAdaptor
	lv         []interface{}
	PhyDevsCnt int
	Capacity   int
}

func GetCommand(args ...string) string {
	bin := "/opt/MegaRAID/MegaCli/MegaCli64"
	cmd := []string{bin}
	cmd = append(cmd, args...)
	return strings.Join(cmd, " ")
}

func GetCommand2(args ...string) string {
	bin := "/opt/MegaRAID/storcli/storcli64"
	cmd := []string{bin}
	cmd = append(cmd, args...)
	return strings.Join(cmd, " ")
}

func (raid *MegaRaid) GetModules() []string {
	ret := []string{}
	lines, err := raid.term.Run("/sbin/lsmod")
	if err != nil {
		log.Errorf("Remote lsmod error: %v", err)
		return ret
	}
	for _, line := range lines {
		mod := line[:strings.Index(line, " ")]
		if mod != "Module" {
			ret = append(ret, mod)
		}
	}
	return ret
}

func (raid *MegaRaid) ParsePhyDevs() bool {
	if !utils.IsInStringArray("megaraid_sas", raid.GetModules()) {
		log.Warningf("Not found megaraid_sas module")
		return false
	}
	cmd := GetCommand("-PDList", "-aALL")
	ret, err := raid.term.Run(cmd)
	if err != nil {
		log.Errorf("List raid disk error: %v", err)
		return false
	}
	err := raid.parsePhyDevs(ret)
	if err != nil {
		log.Errorf("parse physical disk device error: %v", err)
		return false
	}
	return true
}

func (raid *MegaRaid) parsePhyDevs(lines []string) error {
	phyDev := NewMegaRaidPhyDev()
	var adapter *MegaRaidAdaptor
	for _, line := range lines {
		matches := adapterPatter.FindStringSubmatch(line)
		if len(matches) != 0 {
			paramsMap := make(map[string]string)
			for i, name := range sizePattern.SubexpNames() {
				if i > 0 && i <= len(matches) {
					paramsMap[name] = matches[i]
				}
			}
			adapterStr := paramsMap["idx"]
			adapterInt, _ := strconv.Atoi(adapterStr)
			adapter = NewMegaRaidAdaptor(adapterInt, raid)
			raid.adapters = append(raid.adapters, adapter)
		} else if phyDev.parseLine(line) && phyDev.isComplete() {
			if adapter == nil {
				return fmt.Errorf("Adapter is nil")
			}
			adapter.AddPhyDev(phyDev)
			raid.PhyDevsCnt += 1
			raid.Capacity += phyDev.GetSize()
			phyDev = NewMegaRaidPhyDev()
		}
	}
	//map(self.add_phydev_strpsz, adapter.devs)
	return nil
}

func (raid *MegaRaid) addPhyDevStripSize(phyDev *MegaRaidPhyDev) bool {
	grepCmd := []string{"grep", "-iE", "'^(Min|Max) Strip Size'"}
	args := []string{"-adpallinfo", "-aall", "|"}
	args = append(args, grepCmd...)
	cmd := GetCommand(args...)
	ret, err := raid.term.Run(cmd)
	if err != nil {
		log.Errorf("addPhyDevStripSize error: %v", err)
		return false
	}
	return phyDev.parseStripSize(ret)
}

func (raid *MegaRaid) GetPhyDevs() []*MegaRaidPhyDev {
	devs := make([]*MegaRaidPhyDev, 0)
	for _, ada := range raid.adapters {
		devs = append(devs, ada.GetPhyDevs()...)
	}
	return devs
}
