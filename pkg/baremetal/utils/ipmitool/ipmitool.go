package ipmitool

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/baremetal/profiles"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/ssh"
	stage_stringutils "yunion.io/x/onecloud/pkg/util/stringutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type IPMIParser struct{}

func (parser *IPMIParser) GetDefaultTimeout() time.Duration {
	return 20 * time.Second
}

var (
	BOOTDEVS = []string{"pxe", "disk", "safe", "diag", "cdrom", "bios"}
	SOLOPTS  = []string{"default", "skip", "enable"}
)

type Args []string

func newArgs(args ...interface{}) Args {
	ret := make([]string, len(args))
	for i, arg := range args {
		ret[i] = fmt.Sprintf("%v", arg)
	}
	return ret
}

type IPMIExecutor interface {
	GetMode() string
	ExecuteCommand(args ...string) ([]string, error)
}

type SSHIPMI struct {
	IPMIParser
	sshClient *ssh.Client
}

func NewSSHIPMI(cli *ssh.Client) *SSHIPMI {
	return &SSHIPMI{
		sshClient: cli,
	}
}

func (ipmi *SSHIPMI) GetMode() string {
	return "ssh"
}

func (ipmi *SSHIPMI) GetCommand(args ...string) *procutils.Command {
	nArgs := []string{"-I", "open"}
	nArgs = append(nArgs, args...)
	return procutils.NewCommand("/usr/bin/ipmitool", nArgs...)
}

func (ipmi *SSHIPMI) ExecuteCommand(args ...string) ([]string, error) {
	cmd := ipmi.GetCommand(args...)
	log.Debugf("[SSHIPMI] execute command: %s", cmd)
	return ipmi.sshClient.Run(cmd.String())
}

type LanPlusIPMI struct {
	IPMIParser
	host     string
	user     string
	password string
	port     int
}

func NewLanPlusIPMI(host, user, password string) *LanPlusIPMI {
	return NewLanPlusIPMIWithPort(host, user, password, 623)
}

func NewLanPlusIPMIWithPort(host, user, password string, port int) *LanPlusIPMI {
	return &LanPlusIPMI{
		host:     host,
		user:     user,
		password: password,
		port:     port,
	}
}

func (ipmi *LanPlusIPMI) GetMode() string {
	return "rmcp"
}

func (ipmi *LanPlusIPMI) GetCommand(args ...string) *procutils.Command {
	nArgs := []string{
		"--signal=KILL",
		fmt.Sprintf("%s", ipmi.GetDefaultTimeout()),
		"ipmitool", "-I", "lanplus", "-H", ipmi.host,
		"-p", fmt.Sprintf("%d", ipmi.port),
		"-U", ipmi.user,
		"-P", ipmi.password,
	}
	nArgs = append(nArgs, args...)
	return procutils.NewCommand("timeout", nArgs...)
}

func (ipmi *LanPlusIPMI) ExecuteCommand(args ...string) ([]string, error) {
	cmd := ipmi.GetCommand(args...)
	log.Debugf("[LanPlusIPMI] execute command: %s", cmd.String())
	out, err := cmd.Run()
	if err != nil {
		return nil, err
	}
	return ssh.ParseOutput(out), nil
}

func GetSysInfo(exector IPMIExecutor) (*types.IPMISystemInfo, error) {
	// TODO: do cache
	args := []string{"fru", "print", "0"}
	lines, err := exector.ExecuteCommand(args...)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]string)

	keys := map[string]string{
		"manufacture": "Product Manufacturer",
		"model":       "Product Name",
		"bmodel":      "Board Product",
		"version":     "Product Version",
		"sn":          "Product Serial",
		"bsn":         "Board Serial",
	}

	for _, line := range lines {
		key, val := stringutils.SplitKeyValue(line)
		if key != "" {
			for n, v := range keys {
				if _, ok := ret[n]; v == key && !ok {
					ret[n] = val
				}
			}
		}
	}
	_, snOk := ret["sn"]
	bsn, bsnOk := ret["bsn"]
	if !snOk && bsnOk {
		// no product serial
		ret["sn"] = bsn
	}
	info := types.IPMISystemInfo{}
	err = sysutils.DumpMapToObject(ret, &info)
	return &info, err
}

func GetLanChannels(sysinfo *types.IPMISystemInfo) []int {
	return profiles.GetLanChannel(sysinfo)
}

func GetDefaultLanChannel(sysinfo *types.IPMISystemInfo) int {
	return GetLanChannels(sysinfo)[0]
}

func GetRootId(sysinfo *types.IPMISystemInfo) int {
	return profiles.GetRootId(sysinfo)
}

func GetLanConfig(exector IPMIExecutor, channel int) (*types.IPMILanConfig, error) {
	args := newArgs("lan", "print", channel)
	lines, err := ExecuteCommands(exector, args)
	if err != nil {
		return nil, err
	}
	ret := new(types.IPMILanConfig)
	for _, line := range lines {
		key, val := stringutils.SplitKeyValue(line)
		if key == "" {
			continue
		}
		switch key {
		case "IP Address Source":
			if val == "Static Address" {
				ret.IPSrc = "static"
			}
		case "IP Address":
			ret.IPAddr = val
		case "Subnet Mask":
			ret.Netmask = val
		case "MAC Address":
			ret.Mac, _ = net.ParseMAC(val)
		case "Default Gateway IP":
			ret.Gateway = val
		}
	}
	return ret, nil
}

func tryExecuteCommand(exector IPMIExecutor, args ...string) ([]string, error) {
	var err error
	var ret []string
	maxTries := 3
	for tried := 0; tried < maxTries; tried++ {
		ret, err = exector.ExecuteCommand(args...)
		if err == nil {
			return ret, nil
		}
		sleepTime := time.Second * (1 << uint(tried))
		log.Errorf("Execute args %v error: %v, sleep %s then try again", args, err, sleepTime)
		time.Sleep(sleepTime)
	}
	return ret, err
}

func ExecuteCommands(exector IPMIExecutor, args ...Args) ([]string, error) {
	results := make([]string, 0)
	for _, arg := range args {
		ret, err := tryExecuteCommand(exector, arg...)
		if err != nil {
			return nil, err
		}
		results = append(results, ret...)
	}
	return results, nil
}

func doActions(exector IPMIExecutor, actionName string, args ...Args) error {
	_, err := ExecuteCommands(exector, args...)
	if err != nil {
		return fmt.Errorf("Do %s action error: %v", actionName, err)
	}
	return nil
}

func SetLanDHCP(exector IPMIExecutor, lanChannel int) error {
	args := newArgs("lan", "set", lanChannel, "ipsrc", "dhcp")
	return doActions(exector, "set_lan_dhcp", args)
}

func SetLanStatic(
	exector IPMIExecutor,
	channel int,
	ip string,
	mask string,
	gateway string,
) error {
	config, err := GetLanConfig(exector, channel)
	if err != nil {
		return err
	}
	var argss []Args
	if config.IPAddr == ip && config.Netmask == mask && config.Gateway == gateway {
		argss = []Args{
			newArgs("lan", "set", channel, "ipsrc", "static"),
			newArgs("lan", "set", channel, "ipaddr", ip),
			newArgs("lan", "set", channel, "netmask", mask),
			newArgs("lan", "set", channel, "defgw", "ipaddr", gateway),
		}
	} else {
		argss = []Args{
			newArgs("lan", "set", channel, "ipaddr", ip),
			newArgs("lan", "set", channel, "defgw", "ipaddr", gateway),
			newArgs("lan", "set", channel, "netmask", mask),
			newArgs("lan", "set", channel, "ipsrc", "static"),
		}
	}
	return doActions(exector, "set_lan_static", argss...)
}

func SetLanStaticIP(exector IPMIExecutor, channel int, ip string) error {
	args := newArgs("lan", "set", channel, "ipaddr", ip)
	return doActions(exector, "set_lan_static_ip", args)
}

func setLanAccess(exector IPMIExecutor, channel int, access string) error {
	args := []Args{
		newArgs("lan", "set", channel, "access", access),
		// newArgs("lan", "set", channel, "auth", "ADMIN", "MD5"),
	}
	return doActions(exector, "set_lan_access", args...)
}

func EnableLanAccess(exector IPMIExecutor, channel int) error {
	return setLanAccess(exector, channel, "on")
}

func SetLanUserPasswd(exector IPMIExecutor, channel int, user string, password string) error {
	sysInfo, err := GetSysInfo(exector)
	if err != nil {
		return err
	}
	password, err = stage_stringutils.EscapeEchoString(password)
	if err != nil {
		return fmt.Errorf("EscapeEchoString for password: %s, error: %v", password, err)
	}
	rootId := GetRootId(sysInfo)
	args := []Args{
		newArgs("user", "enable", rootId),
		newArgs("user", "set", "name", rootId, user, fmt.Sprintf("\"%s\"", password)),
		newArgs("user", "priv", rootId, 4, channel),
	}
	err = doActions(exector, "set_lan_user_password", args...)
	if err != nil {
		return err
	}
	args = []Args{newArgs(
		"raw", "0x06", "0x43",
		fmt.Sprintf("0x%02x", 0xb0+channel),
		fmt.Sprintf("0x%02x", rootId), "0x04", "0x00")}
	err = doActions(exector, "set_lan_user_password2", args...)
	if err == nil {
		return nil
	}
	args = []Args{newArgs(
		"channel", "setaccess", channel,
		rootId, "link=on", "ipmi=on",
		"callin=on", "privilege=4",
	)}
	return doActions(exector, "set_lan_user_password3", args...)
}

func SetLanPasswd(exector IPMIExecutor, rootId int, password string) error {
	// TODO: escape password
	args := newArgs("user", "set", "password", rootId, fmt.Sprint("\"%s\"", password))
	return doActions(exector, "set_lan_passwd", args)
}

func GetChassisPowerStatus(exector IPMIExecutor) (string, error) {
	args := newArgs("chassis", "power", "status")
	ret, err := ExecuteCommands(exector, args)
	if err != nil {
		return "", err
	}
	for _, line := range ret {
		if strings.Contains(line, "Chassis Power is") {
			data := strings.Split(line, " ")
			status := strings.ToLower(strings.TrimSpace(data[len(data)-1]))
			return status, nil
		}
	}
	return "", fmt.Errorf("Unknown chassis status")
}

func GetBootFlags(exector IPMIExecutor) (*types.IPMIBootFlags, error) {
	args := newArgs("raw", "0x00", "0x09", "0x05", "0x00", "0x00")
	ret, err := ExecuteCommands(exector, args)
	if err != nil {
		return nil, err
	}
	bytes, err := HexStr2Bytes(ret[0])
	if err != nil {
		return nil, err
	}
	bootdevIdx := ((bytes[3] >> 2) & 0x0f) - 1
	bootdev := ""
	if bootdevIdx >= 0 && int(bootdevIdx) < len(BOOTDEVS) {
		bootdev = BOOTDEVS[bootdevIdx]
	}
	flags := &types.IPMIBootFlags{
		Dev: bootdev,
	}
	solIdx := (bytes[4] & 0x03)
	if solIdx == 1 {
		sol := false
		flags.Sol = &sol
	} else if solIdx == 2 {
		sol := true
		flags.Sol = &sol
	}
	log.Errorf("====bytes: %v, flags: %#v", bytes, flags)
	return flags, nil
}

func HexStr2Bytes(hs string) ([]int64, error) {
	log.Errorf("======HexStr2Bytes: %#v", hs)
	b := []int64{}
	for _, x := range strings.Split(hs, " ") {
		intV, err := strconv.ParseInt(x, 16, 64)
		if err != nil {
			return nil, err
		}
		b = append(b, intV)
	}
	return b, nil
}

func GetACPIPowerStatus(exector IPMIExecutor) ([]int64, error) {
	args := newArgs("raw", "0x06", "0x07")
	ret, err := ExecuteCommands(exector, args)
	if err != nil {
		return nil, err
	}
	return HexStr2Bytes(ret[0])
}

func DoSoftShutdown(exector IPMIExecutor) error {
	args := newArgs("chassis", "power", "soft")
	return doActions(exector, "do_soft_shutdown", args)
}

func DoHardShutdown(exector IPMIExecutor) error {
	args := newArgs("chassis", "power", "off")
	return doActions(exector, "do_hard_shutdown", args)
}

func DoPowerOn(exector IPMIExecutor) error {
	args := newArgs("chassis", "power", "on")
	return doActions(exector, "do_power_on", args)
}

func DoPowerReset(exector IPMIExecutor) error {
	args := newArgs("chassis", "power", "reset")
	return doActions(exector, "do_power_reset", args)
}

func DoPowerCycle(exector IPMIExecutor) error {
	args := newArgs("chassis", "power", "cycle")
	return doActions(exector, "do_power_cycle", args)
}

func DoReboot(exector IPMIExecutor) error {
	maxTries := 10

	var status string
	var err error
	status, err = GetChassisPowerStatus(exector)
	if err != nil {
		return err
	}

	isValidStatus := func(s string) bool {
		return utils.IsInStringArray(s, []string{types.POWER_STATUS_ON, types.POWER_STATUS_OFF})
	}

	for tried := 0; !isValidStatus(status) && tried <= maxTries; tried++ {
		time.Sleep(1 * time.Second)
		status, err = GetChassisPowerStatus(exector)
		if err != nil {
			return err
		}
	}

	if !isValidStatus(status) {
		return fmt.Errorf("Unexpected status: %s", status)
	}

	// do shutdown
	if status == types.POWER_STATUS_ON {
		err = DoHardShutdown(exector)
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
		for tried := 0; tried < maxTries; tried++ {
			status, err = GetChassisPowerStatus(exector)
			if err != nil {
				return err
			}
			if status == types.POWER_STATUS_OFF {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}

	// do power on
	status, err = GetChassisPowerStatus(exector)
	if err != nil {
		return err
	}
	for tried := 0; status != types.POWER_STATUS_ON && tried < maxTries; tried++ {
		err = DoPowerOn(exector)
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
		status, err = GetChassisPowerStatus(exector)
		if err != nil {
			return err
		}
	}

	status, err = GetChassisPowerStatus(exector)
	if err != nil {
		return err
	}
	if status != types.POWER_STATUS_ON {
		return fmt.Errorf("do reboot fail to poweron, current status: %s", status)
	}
	return nil
}

func doRebootToFlag(exector IPMIExecutor, setFunc func(IPMIExecutor) error) error {
	err := setFunc(exector)
	if err != nil {
		return err
	}
	return DoReboot(exector)
}

func SetRebootToDisk(exector IPMIExecutor) error {
	return SetBootFlags(exector, "disk", tristate.True, true)
}

func DoRebootToDisk(exector IPMIExecutor) error {
	return doRebootToFlag(exector, SetRebootToDisk)
}

func SetRebootToPXE(exector IPMIExecutor) error {
	return SetBootFlagPXE(exector)
}

func DoRebootToPXE(exector IPMIExecutor) error {
	return doRebootToFlag(exector, SetRebootToPXE)
}

func SetRebootToBIOS(exector IPMIExecutor) error {
	return SetBootFlags(exector, "bios", tristate.True, false)
}

func DoRebootToBIOS(exector IPMIExecutor) error {
	return doRebootToFlag(exector, SetRebootToBIOS)
}

func SetBootFlagPXE(exector IPMIExecutor) error {
	return setBootFlagsV2(exector, "pxe")
}

func SetBootFlags(
	exector IPMIExecutor,
	bootdev string,
	sol tristate.TriState,
	enablePersistent bool,
) error {
	err := setBootFlagsV1(exector, bootdev, sol, enablePersistent)
	if err == nil {
		return nil
	}
	return setBootFlagsV2(exector, bootdev)
}

func setBootFlagsV1(
	exector IPMIExecutor,
	bootdev string,
	sol tristate.TriState,
	enablePersistent bool,
) error {
	cmd := []interface{}{"raw", "0x00", "0x08", "0x05"}
	bootdevIdx := 0
	if ok, idx := utils.InStringArray(bootdev, BOOTDEVS); ok {
		bootdevIdx = idx + 1
	} else {
		return fmt.Errorf("Illegal bootdev %s", bootdev)
	}
	valid := 0x80
	if enablePersistent {
		valid = valid + 0x40
	}
	solIdx := 0
	if !sol.IsNone() {
		if sol.IsTrue() {
			solIdx = 2
		} else {
			solIdx = 1
		}
	}
	for _, x := range []int{valid, bootdevIdx << 2, solIdx, 0, 0} {
		cmd = append(cmd, fmt.Sprintf("0x%02x", x))
	}
	return doActions(exector, "set_boot_flags_v1", newArgs(cmd...))
}

func setBootFlagsV2(exector IPMIExecutor, bootdev string) error {
	return doActions(
		exector,
		fmt.Sprintf("set_boot_flag_%s", bootdev),
		newArgs("chassis", "bootdev", bootdev),
	)
}

func GetIPMILanPort(exector IPMIExecutor) (string, error) {
	ret, err := ExecuteCommands(exector, newArgs("delloem", "lan", "get"))
	if err != nil {
		return "", err
	}
	return ret[1], nil
}

func SetDellIPMILanPortShared(exector IPMIExecutor) error {
	args1 := newArgs("delloem", "lan", "set", "shared")
	args2 := newArgs("delloem", "lan", "set", "shared", "with", "lom1")
	err2 := doActions(exector, "_dell_set_ipmi_lan_port_shared_02", args2)
	if err2 != nil {
		return doActions(exector, "_dell_set_ipmi_lan_port_shared_01", args1)
	}
	return nil
}

func SetHuaweiIPMILanPortShared(exector IPMIExecutor) error {
	args := []Args{
		newArgs(
			"raw", "0xc", "0x1", "0x1", "0xd7", "0xdb",
			"0x07", "0x00", "0x2",
		),
		newArgs(
			"raw", "0x30", "0x93", "0xdb", "0x07", "0x00",
			"0x05", "0x0d", "0x0", "0x0", "0x1", "0x0",
		),
	}
	return doActions(exector, "_huawei_set_ipmi_lan_port_shared", args...)
}

func SetIPMILanPortDedicated(exector IPMIExecutor) error {
	return doActions(
		exector,
		"set_ipmi_lan_port_dedicated",
		newArgs("delloem", "lan", "set", "dedicated"),
	)
}

func DoBMCReset(exector IPMIExecutor) error {
	return doActions(exector, "do_bmc_reset", newArgs("mc", "reset", "cold"))
}
