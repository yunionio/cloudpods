package megactl

import (
	"fmt"
	"regexp"
	"sort"
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
	devs  []*baremetal.BaremetalStorage
	lvs   []interface{}
}

func NewMegaRaidAdaptor(index int, raid *MegaRaid) *MegaRaidAdaptor {
	return &MegaRaidAdaptor{
		index: index,
		raid:  raid,
	}
}

func (adapter *MegaRaidAdaptor) getTerm() *ssh.Client {
	return adapter.raid.term
}

func (adapter *MegaRaidAdaptor) remoteRun(cmds ...string) ([]string, error) {
	return adapter.getTerm().Run(cmds...)
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

func (adapter *MegaRaidAdaptor) BuildRaid(confs []*baremetal.BaremetalDiskConfig) bool {
	adapter.clearJBODDisks()
	if !adapter.removeLogicVolumes() {
		return false
	}
	if len(adapter.devs) == 0 {
		// no disk to build
		return true
	}
	var left []*baremetal.BaremetalStorage = adapter.devs
	var selected []*baremetal.BaremetalStorage
	for _, conf := range confs {
		selected, left = baremetal.RetrieveStorages(conf, left)
		if len(selected) == 0 {
			log.Errorf("No enough disks for config %#v", conf)
			return false
		}
		result := true
		switch conf.Conf {
		case baremetal.DISK_CONF_RAID5:
			result = adapter.buildRaid(selected, conf, "5")
		case baremetal.DISK_CONF_RAID10:
			result = adapter.buildRaid(selected, conf, "10")
		}
	}
}

func (adapter *MegaRaidAdaptor) conf2ParamsStorcliSize(conf *baremetal.BaremetalDiskConfig) []string {
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

func (adapter *MegaRaidAdaptor) conf2ParamsStorcli(conf *baremetal.BaremetalDiskConfig) []string {
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

func conf2Params(conf *baremetal.BaremetalDiskConfig) []string {
	params := []string{}
	if conf.WT != nil {
		if *conf.WT {
			params = append(params, "WT")
		} else {
			params = append(params, "WB")
		}
	}
	if conf.RA != nil {
		if *conf.RA {
			params = append(params, "RA")
		} else {
			params = append(params, "NORA")
		}
	}
	if conf.Direct != nil {
		if *conf.Direct {
			params = append(params, "Direct")
		} else {
			params = append(params, "Cached")
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
		params = append(params, fmt.Sprintf("-strpsz%d", *conf.Strip))
	}
	if len(conf.Size) > 0 {
		for _, sz := range conf.Size {
			params = append(params, fmt.Sprintf("-sz%d", sz))
		}
	}
	return params
}

func (adapter *MegaRaidAdaptor) _storcliBuildRaid0(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter._storcliBuildRaid(devs, conf, 0)
}

func (adapter *MegaRaidAdaptor) _megacliBuildRaid0(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter._megacliBuildRaid(devs, conf, 0)
}

func (adapter *MegaRaidAdaptor) _storcliBuildRaid1(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter._storcliBuildRaid(devs, conf, 1)
}

func (adapter *MegaRaidAdaptor) _megacliBuildRaid1(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter._megacliBuildRaid(devs, conf, 1)
}

func (adapter *MegaRaidAdaptor) _storcliBuildRaid5(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter._storcliBuildRaid(devs, conf, 5)
}

func (adapter *MegaRaidAdaptor) _megacliBuildRaid5(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter._megacliBuildRaid(devs, conf, 5)
}

func (adapter *MegaRaidAdaptor) _storcliBuildRaid10(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	return adapter._storcliBuildRaid(devs, conf, 10)
}

func (adapter *MegaRaidAdaptor) _megacliBuildRaid10(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	if len(devs)%2 != 0 {
		return fmt.Errorf("Odd number of %d devs", len(devs))
	}
	devCnt := len(devs) / 2
	params := []string{}
	for i := 0; i < devCnt; i++ {
		d1 := devs[i]
		d2 = devs[i+devCnt]
		params = append(params, fmt.Sprintf("-Array%d[%s,%s]", i, d1.String(), d2.String()))
	}
	args = []string{"-CfgSpanAdd", "-r10"}
	args = append(args, params...)
	args = append(args, conf2Params(conf))
	args = append(args, fmt.Sprintf("-a%d", adapter.index))
	cmd := GetCommand(args...)
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) _storcliBuildRaid(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig, level uint) error {
	args := []string{}
	args = append(args, fmt.Sprintf("/c%d", adapter.index))
	args = append(args, "add", "vd", fmt.Sprintf("type=r%d", level))
	args = append(args, adapter.conf2ParamsStorcliSize(conf)...)
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, dev.String())
	}
	args = append(args, fmt.Sprintf("drives=%s", strings.Join(labels, ",")))
	if level == 10 {
		args = append(args, "PDperArray=2")
	}
	args = append(args, adapter.conf2ParamsStorcli(conf)...)
	cmd := GetCommand2(args...)
	log.Infof("_storcliBuildRaid command: %s", cmd)
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) _megacliBuildRaid(devs []*baremtal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig, level uint) error {
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, dev.String())
	}
	args := []string{"-CfgLdAdd", fmt.Sprintf("-r%d", level), fmt.Sprintf("[%s]", strings.Join(labels, ","))}
	args = append(args, conf2Params(conf)...)
	args = append(args, fmt.Sprintf("-a%d", adapter.index))
	cmd := GetCommand(args...)
	log.Infof("_megacliBuildRaid command: %s", cmd)
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) buildRaid(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig, level string) bool {

}

func (adapter *MegaRaidAdaptor) storcliIsJBODEnabled() bool {
	cmd := GetCommand2(fmt.Sprintf("/c%d", adapter.index), "show", "jbod")
	lines, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("storcliIsJBODEnabled error: %s", err)
		return false
	}
	for _, line := range lines {
		line = strings.ToLower(line)
		if strings.HasPrefix(line, "jbod") {
			data := strings.Split(line, " ")
			if strings.TrimSpace(data[1]) == "on" {
				return true
			}
			return false
		}
	}
	return false
}

func (adapter *MegaRaidAdaptor) storcliEnableJBOD(enable bool) bool {
	val := "off"
	if enable {
		val = "on"
	}
	cmd := GetCommand2(fmt.Sprintf("/c%d", adapter.index), "set", fmt.Sprintf("jbod=%s", val))
	_, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("EnableJBOD %v fail: %v", enable, err)
		return false
	}
	return true
}

func (adapter *MegaRaidAdaptor) storcliBuildJBOD(devs []*baremetal.BaremetalStorage) error {
	if !adapter.storcliIsJBODEnabled() {
		adapter.storcliEnableJBOD(true)
		adapter.storcliEnableJBOD(false)
		adapter.storcliEnableJBOD(true)
	}
	if !adapter.storcliIsJBODEnabled() {
		return fmt.Errorf("JBOD not supported")
	}
	cmds := []string{}
	for _, d := range devs {
		cmd := GetCommand2(fmt.Sprintf("/c%d/e%s/s%d", adapter.index, d.Enclosure, d.Slot))
		cmds = append(cmds, cmd)
	}
	log.Infof("storcliBuildJBOD cmds: %v", cmds)
	_, err := adapter.remoteRun(cmds...)
	if err != nil {
		return err
	}
	return nil
}

func (adapter *MegaRaidAdaptor) storcliBuildNoRaid(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	err := adapter.storcliBuildJBOD(devs)
	if err == nil {
		return nil
	}
	log.Errorf("Try build JBOD fail: %v", err)
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, dev.String())
	}
	args := []string{
		fmt.Sprintf("/c%d", adapter.index),
		"add", "vd", "each", "type=raid0",
		fmt.Sprintf("drives=%s", strings.Join(labels, ",")),
		"wt", "nora", "direct",
	}
	cmd := GetCommand2(args...)
	_, err = adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) megacliBuildNoRaid(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error {
	err := adapter.megacliBuildJBOD()
	if err == nil {
		return nil
	}
	log.Errorf("Try build jbod fail: %v", err)
	cmds := []string{}
	for _, d := range devs {
		cmd = GetCommand("-CfgLdAdd", "-r0", fmt.Sprintf("[%s]", d.String()),
			"WT", "NORA", "Direct", "NoCachedBadBBU", fmt.Sprintf("-a%d", adapter.index))
	}
	_, err = adapter.remoteRun(cmds...)
	return err
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

func (adapter *MegaRaidAdaptor) megacliBuildJBOD(devs []*baremetal.BaremetalStorage) error {
	if !adapter.megacliIsJBODEnabled() {
		adapter.megacliEnableJBOD(true)
		adapter.megacliEnableJBOD(false)
		adapter.megacliEnableJBOD(true)
	}
	if !adapter.megacliIsJBODEnabled() {
		return fmt.Errorf("JBOD not supported")
	}
	devIds := []string{}
	for _, d := range devs {
		devIds = append(devIds, fmt.Sprintf("%s", d.String()))
	}
	cmd = GetCommand("-PDMakeJBOD", fmt.Sprintf("-PhysDrv[%s]", strings.Join(devIds, ",")), fmt.Sprintf("-a%d", adapter.index))
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) removeLogicVolumes() bool {
	cmds := []string{}
	lvIdx := adapter.getLogicVolumes()
	for i := len(lvIdx) - 1; i >= 0; i-- {
		cmd := GetCommand("-CfgLdDel", fmt.Sprintf("-L%d", i), "-Force", fmt.Sprintf("-a%d", adapter.index))
		cmds = append(cmds, cmd)
	}
	if len(cmds) > 0 {
		_, err := adapter.remoteRun(cmds...)
		if err != nil {
			return false
		}
		return true
	}
	return true
}

/*
def _storcli_clear_jbod_disks(self):
    cmds = []
    for dev in self.devs:
        cmd = self.raid.get_command2(
                    '/c%d/e%d/s%d' % (self.index, dev.enclosure, dev.slot),
                    'set', 'good', 'force')
        cmds.append(cmd)
    logging.info('%s', cmds)
    self.raid.term.exec_remote_commands(cmds)
*/

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

//func (raid *MegaRaid) ParseLogicVolumes() bool {
//cmd := GetCommand("-LDInfo", "-Lall", "-aALL")
//ret, err := raid.term.Run(cmd)
//if err != nil {
//return false
//}
//raid.parseLogicVolumes(ret)
//return true
//}

func (raid *MegaRaid) CleanRaid() {
	for _, adapter := range raid.adapters {
		adapter.clearJBODDisks()
		adapter.removeLogicVolumes()
	}
}

func (raid *MegaRaid) BuildRaid() {

}

func (raid *MegaRaid) GetAdapter(index int) *MegaRaidAdaptor {
	for _, adapter := range raid.adapters {
		if adapter.index == index {
			return adapter
		}
	}
	return nil
}

func (raid *MegaRaid) RemoveLogicVolumes() {
	for _, adapter := range raid.adapters {
		adapter.removeLogicVolumes()
	}
}

func (raid *MegaRaid) GetLogicVolumes() []int {
	lvs := []int{}
	for _, adapter := range raid.adapters {
		lvs = append(lvs, adapter.getLogicVolumes()...)
	}
	return lvs
}
