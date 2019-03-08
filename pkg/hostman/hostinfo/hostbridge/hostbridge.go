package hostbridge

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/system_service"
	"yunion.io/x/onecloud/pkg/util/bwutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/ovsutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type IBridgeDriver interface {
	ConfirmToConfig(bool, []string) (bool, error)
	Setup() error
	Exists() bool
	Interfaces() []string

	FetchConfig()
	GetMac() string
	GenerateIfupScripts(scriptPath string, nic jsonutils.JSONObject) error
	GenerateIfdownScripts(scriptPath string, nic jsonutils.JSONObject) error
	RegisterHostlocalServer(mac, ip string) error
	WarmupConfig() error
	CleanupConfig()
}

type SBaseBridgeDriver struct {
	bridge *netutils2.SNetInterface
	ip     string
	inter  *netutils2.SNetInterface
}

func NewBaseBridgeDriver(bridge, inter, ip string) (*SBaseBridgeDriver, error) {
	var bd = new(SBaseBridgeDriver)
	bd.bridge = netutils2.NewNetInterface(bridge)
	if len(inter) > 0 {
		bd.inter = netutils2.NewNetInterface(inter)
		if !bd.inter.Exist() {
			return nil, fmt.Errorf("%s not exists", inter)
		}
		bd.ip = ip
		bd.inter.DisableGso()
	} else if len(ip) > 0 {
		return nil, fmt.Errorf("A bridge without interface must have no IP")
	}
	return bd, nil
}

func (d *SBaseBridgeDriver) FetchConfig() {
	d.bridge.FetchConfig()
	d.inter.FetchConfig()
}

func (d *SBaseBridgeDriver) GetMac() string {
	if len(d.bridge.Mac) == 0 {
		d.bridge.FetchConfig()
	}
	return d.bridge.Mac
}

func (d *SBaseBridgeDriver) BringupInterface() error {
	var infs = []*netutils2.SNetInterface{d.bridge}
	if d.inter != nil {
		infs = append(infs, d.inter)
	}
	for _, inf := range infs {
		cmd := []string{"ifconfig", inf.String(), "up"}
		if options.HostOptions.TunnelPaddingBytes > 0 {
			cmd = append(cmd, "mtu", fmt.Sprintf("%d", options.HostOptions.TunnelPaddingBytes))
		}
		if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Run(); err != nil {
			return err
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) ConfirmToConfig(exists bool, infs []string) (bool, error) {
	if exists {
		d.bridge.FetchConfig()
		if len(d.ip) > 0 {
			if len(d.bridge.Addr) == 0 {
				if len(d.inter.Addr) == 0 {
					return false, fmt.Errorf("Neither %s nor %s owner address %s",
						d.inter, d.bridge, d.ip)
				}
				if d.inter.Addr != d.ip {
					return false, fmt.Errorf("%s!=%s, %s not same as config",
						d.ip, d.inter.Addr, d.inter)
				}
				log.Infof("Bridge address is not configured")
				return false, nil
			}
			if d.bridge.Addr != d.ip {
				return false, fmt.Errorf("%s IP %s!=%s, mismatch", d.bridge, d.bridge.Addr, d.ip)
			}
		} else {
			if d.inter != nil && len(d.inter.Addr) > 0 {
				return false, fmt.Errorf("%s should have no address", d.inter)
			}
			if len(d.bridge.Addr) == 0 {
				return false, nil
			}
			if !d.bridge.IsSecretInterface() {
				return false, fmt.Errorf("%s should have address in 169.254.0.0/16", d.bridge)
			}
		}
		if d.inter != nil && !utils.IsInStringArray(d.inter.String(), infs) {
			log.Infof("Interface %s not in bridge...", d.inter)
			return false, nil
		}
		if err := d.BringupInterface(); err != nil {
			log.Errorln(err)
			return false, err
		}
		return true, nil
	} else {
		if d.bridge.FetchInter() != nil {
			return false, fmt.Errorf("Bridge %s exists, but not created by this driver????", d.bridge)
		}
		if len(d.ip) > 0 && (d.inter == nil || len(d.inter.Addr) == 0) {
			return false, fmt.Errorf("Interface %s not configured", d.inter)
		}
		return false, nil
	}
}

func (d *SBaseBridgeDriver) SetupAddresses(mask net.IPMask) error {
	var addr string
	if len(d.ip) == 0 {
		addr, mask = netutils2.GetSecretInterfaceAddress()
	} else {
		addr = d.ip
	}
	cmd := []string{"ifconfig", d.bridge.String(), addr, "netmask", netutils2.NetBytes2Mask(mask)}
	if options.HostOptions.TunnelPaddingBytes > 0 {
		cmd = append(cmd, "mtu", fmt.Sprintf("%d", options.HostOptions.TunnelPaddingBytes+1500))
	}
	if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Run(); err != nil {
		log.Errorln(err)
		return fmt.Errorf("Failed to bring up bridge %s", d.bridge)
	}
	if d.inter != nil {
		if _, err := procutils.NewCommand("ifconfig", d.inter.String(), "0", "up").Run(); err != nil {
			log.Errorln(err)
			return fmt.Errorf("Failed to bring up interface %s", d.inter)
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) SetupSlaveAddresses(slaveAddrs [][]string) error {
	for _, slaveAddr := range slaveAddrs {
		cmd := []string{"ip", "address", "del",
			fmt.Sprintf("%s/%s", slaveAddr[0], slaveAddr[1]), "dev", d.inter.String()}
		if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Run(); err != nil {
			log.Errorln("Failed to remove slave address from interface %s: %s", d.inter, err)
		}

		cmd = []string{"ip", "address", "add",
			fmt.Sprintf("%s/%s", slaveAddr[0], slaveAddr[1]), "dev", d.bridge.String()}
		if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("Failed to remove slave address from interface %s: %s", d.bridge, err)
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) SetupRoutes(routes [][]string) error {
	for _, r := range routes {
		var cmd []string
		if r[2] == "0.0.0.0" {
			cmd = []string{"route", "add", "default", "gw", r[1], "dev", d.bridge.String()}
		} else {
			cmd = []string{"route", "add", "-net", r[0], "netmask", r[2], "gw", r[1], "dev", d.bridge.String()}
		}
		if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Run(); err != nil {
			log.Errorln(err)
			return fmt.Errorf("Failed to add slave address to bridge %s", d.bridge)
		}
	}
	return nil
}

type SOVSBridgeDriver struct {
	SBaseBridgeDriver
}

func (o *SOVSBridgeDriver) CleanupConfig() {
	ovsutils.CleanAllHiddenPorts()
	// if enableopenflowcontroller ...
}

func (o *SOVSBridgeDriver) Exists() bool {
	data, err := procutils.NewCommand("ovs-vsctl", "list-br").Run()
	if err != nil {
		log.Errorln(err)
		return false
	}
	for _, d := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(d) == o.bridge.String() {
			return true
		}
	}
	return false
}

func (o *SOVSBridgeDriver) Interfaces() []string {
	data, err := procutils.NewCommand("ovs-vsctl", "list-ifaces", o.bridge.String()).Run()
	if err != nil {
		log.Errorln(err)
		return nil
	}

	var infs = make([]string, 0)
	for _, d := range strings.Split(string(data), "\n") {
		if len(strings.TrimSpace(d)) > 0 {
			infs = append(infs, strings.TrimSpace(d))
		}
	}
	return infs
}

func (o *SOVSBridgeDriver) Setup() error {
	var routes [][]string
	var slaveAddrs [][]string
	if o.inter != nil && len(o.inter.Addr) > 0 {
		routes = o.inter.GetRoutes(true)
		slaveAddrs = o.inter.GetSlaveAddresses()
	}
	if !o.Exists() {
		if err := o.SetupBridgeDev(); err != nil {
			return err
		}
	}

	if o.inter != nil && !utils.IsInStringArray(o.inter.String(), o.Interfaces()) {
		if err := o.SetupInterface(); err != nil {
			return err
		}
	}
	if len(o.bridge.Addr) == 0 {
		if len(o.ip) > 0 {
			if err := o.SetupAddresses(o.inter.Mask); err != nil {
				return err
			}
			if len(slaveAddrs) > 0 {
				if err := o.SetupSlaveAddresses(slaveAddrs); err != nil {
					return err
				}
			}
			if len(routes) > 0 {
				if err := o.SetupRoutes(routes); err != nil {
					return err
				}
			}
		} else {
			if err := o.SetupAddresses(nil); err != nil {
				return err
			}
		}
	}

	return o.BringupInterface()
}

func (o *SOVSBridgeDriver) SetupInterface() error {
	if o.inter != nil && !utils.IsInStringArray(o.inter.String(), o.Interfaces()) {
		output, err := procutils.NewCommand("ovs-vsctl", "--", "--may-exist",
			"add-port", o.bridge.String(), o.inter.String()).Run()
		if err != nil {
			return fmt.Errorf("Failed to add interface %s", output)
		}
	}
	return nil
}

func (o *SOVSBridgeDriver) SetupBridgeDev() error {
	if !o.Exists() {
		_, err := procutils.NewCommand("ovs-vsctl", "--", "--may-exist", "add-br", o.bridge.String()).Run()
		return err
	}
	return nil
}

func (o *SOVSBridgeDriver) GenerateIfdownScripts(scriptPath string, nic jsonutils.JSONObject) error {
	script, err := o.getDownScripts(nic)
	if err != nil {
		log.Errorln(err)
		return err
	}
	return o.saveFileExecutable(scriptPath, script)
}

func (o *SOVSBridgeDriver) GenerateIfupScripts(scriptPath string, nic jsonutils.JSONObject) error {
	script, err := o.getUpScripts(nic)
	if err != nil {
		log.Errorln(err)
		return err
	}
	return o.saveFileExecutable(scriptPath, script)
}

func (o *SOVSBridgeDriver) saveFileExecutable(scriptPath, script string) error {
	if err := fileutils2.FilePutContents(scriptPath, script, false); err != nil {
		return err
	}
	return os.Chmod(scriptPath, syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR)
}

func (o *SOVSBridgeDriver) getUpScripts(nic jsonutils.JSONObject) (string, error) {
	var (
		bridge, _ = nic.GetString("bridge")
		ifname, _ = nic.GetString("ifname")
		ip, _     = nic.GetString("ip")
		mac, _    = nic.GetString("mac")
		vlan, _   = nic.Int("vlan")
	)

	s := "#!/bin/bash\n\n"
	s += fmt.Sprintf("SWITCH='%s'\n", bridge)
	s += fmt.Sprintf("IF='%s'\n", ifname)
	s += fmt.Sprintf("IP='%s'\n", ip)
	s += fmt.Sprintf("MAC='%s'\n", mac)
	s += fmt.Sprintf("VLAN_ID=%d\n", vlan)
	limit, burst, err := bwutils.GetOvsBwValues(nic)
	if err != nil {
		return "", err
	}
	s += fmt.Sprintf("LIMIT=%d\n", limit)
	s += fmt.Sprintf("BURST=%d\n", burst)
	bwDownload, err := bwutils.GetDownloadBwValue(nic, options.HostOptions.BwDownloadBandwidth)
	if err != nil {
		return "", err
	}
	s += fmt.Sprintf("LIMIT_DOWNLOAD='%dmbit'\n", bwDownload)
	if options.HostOptions.TunnelPaddingBytes > 0 {
		s += fmt.Sprintf("/sbin/ifconfig $IF mtu %d\n",
			1500+options.HostOptions.TunnelPaddingBytes)
	}
	s += "/sbin/ifconfig $IF 0.0.0.0 up\n"
	s += "ovs-vsctl list-ifaces $SWITCH | grep -w $IF > /dev/null 2>&1\n"
	s += "if [ $? -eq '0' ]; then\n"
	s += "    ovs-vsctl del-port $SWITCH $IF\n"
	s += "fi\n"
	s += "if [ \"$VLAN_ID\" -ne \"1\" ]; then\n"
	s += "    TAG=\"tag=$VLAN_ID\"\n"
	s += "fi\n"
	s += "ovs-vsctl add-port $SWITCH $IF $TAG\n"
	s += "PORT=$(ovs-ofctl show $SWITCH | grep -w $IF)\n"
	s += "PORT=$(echo $PORT | awk 'BEGIN{FS=\"(\"}{print $1}')\n"
	s += "OFCTL=$(ovs-vsctl get-controller $SWITCH)\n"
	s += "if [ -z \"$OFCTL\" ]; then\n"
	s += "    ovs-vsctl set Interface $IF ingress_policing_rate=$LIMIT\n"
	s += "    ovs-vsctl set Interface $IF ingress_policing_burst=$BURST\n"
	for _, r := range o.GetOfRules(nic) {
		s += "    " + o.AddFlow(r.cond, r.priority, r.actions)
	}
	s += "fi\n"
	s += "if [ $LIMIT_DOWNLOAD != \"0mbit\" ]; then\n"
	s += "    tc qdisc del dev $IF root 2>/dev/null\n"
	s += "    tc qdisc add dev $IF root handle 1: htb default 10\n"
	s += "    tc class add dev $IF parent 1: classid 1:1 htb " +
		"rate $LIMIT_DOWNLOAD ceil $LIMIT_DOWNLOAD\n"
	s += "    tc class add dev $IF parent 1:1 classid 1:10 htb " +
		"rate $LIMIT_DOWNLOAD ceil $LIMIT_DOWNLOAD\n"
	s += "fi\n"
	return s, nil
}

func (o *SOVSBridgeDriver) getDownScripts(nic jsonutils.JSONObject) (string, error) {
	var (
		bridge, _ = nic.GetString("bridge")
		ifname, _ = nic.GetString("ifname")
		ip, _     = nic.GetString("ip")
		mac, _    = nic.GetString("mac")
		vlan, _   = nic.Int("vlan")
	)

	s := "#!/bin/bash\n\n"
	s += fmt.Sprintf("SWITCH='%s'\n", bridge)
	s += fmt.Sprintf("IF='%s'\n", ifname)
	s += fmt.Sprintf("IP='%s'\n", ip)
	s += fmt.Sprintf("MAC='%s'\n", mac)
	s += fmt.Sprintf("VLAN_ID=%d\n", vlan)
	s += "PORT=$(ovs-ofctl show $SWITCH | grep -w $IF)\n"
	s += "if [ $? -ne '0' ]; then\n"
	s += "    exit 0\n"
	s += "fi\n"
	s += "OFCTL=$(ovs-vsctl get-controller $SWITCH)\n"
	s += "PORT=$(echo $PORT | awk 'BEGIN{FS=\"(\"}{print $1}')\n"
	s += "if [ -z \"$OFCTL\" ]; then\n"
	for _, r := range o.GetOfRules(nic) {
		s += "    " + o.DelFlow(r.cond)
	}
	s += "fi\n"
	s += "/sbin/ifconfig $IF 0.0.0.0 down\n"
	s += "ovs-vsctl -- --if-exists del-port $SWITCH $IF\n"
	return s, nil
}

type SRule struct {
	priority int
	cond     string
	actions  string
}

func (o *SOVSBridgeDriver) AddFlow(cond string, priority int, actions string) string {
	s := ""
	s += fmt.Sprintf("ovs-ofctl add-flow $SWITCH \"%s", cond)
	s += fmt.Sprintf(" priority=%d", priority)
	s += fmt.Sprintf(" actions=%s\"\n", actions)
	return s
}

func (o *SOVSBridgeDriver) DoAddFlow(cond string, pri int, actions, swt string) error {
	_, err := procutils.NewCommand("ovs-ofctl", "add-flow", swt,
		fmt.Sprintf("%s priority=%d actions=%s", cond, pri, actions)).Run()
	return err
}

func (o *SOVSBridgeDriver) DelFlow(cond string) string {
	return fmt.Sprintf("ovs-ofctl del-flows $SWITCH \"%s\"\n", cond)
}

func (o *SOVSBridgeDriver) GetOfRules(nic jsonutils.JSONObject) []SRule {
	rules := []SRule{}
	metadataPort := o.GetMetadataServerPort()
	rules = append(rules,
		SRule{9000, fmt.Sprintf("table=0 in_port=local tcp nw_dst=$IP tp_src=%d", metadataPort),
			"mod_nw_src=169.254.169.254,mod_tp_src:80,output:$PORT"},
		SRule{9500, "table=0 in_port=$PORT udp tp_src=68 tp_dst=67", "local"},
		SRule{8000, "table=0 in_port=$PORT", "resubmit(,1)"},
	)
	if vlan, _ := nic.Int("vlan"); vlan != 1 {
		rules = append(rules,
			SRule{4901, "table=1 dl_dst=$MAC,dl_vlan=$VLAN_ID", "strip_vlan,output:$PORT"})
	}
	rules = append(rules,
		SRule{4900, "table=1 dl_dst=$MAC", "output:$PORT"})
	return rules
}

func (o *SOVSBridgeDriver) GetMetadataServerPort() int {
	return options.HostOptions.Port
}

func (o *SOVSBridgeDriver) RegisterHostlocalServer(mac, ip string) error {
	if !options.HostOptions.EnableOpenflowController {
		metadataPort := o.GetMetadataServerPort()
		if err := o.DoAddFlow("table=0 ipv6", 20000, "drop", o.bridge.String()); err != nil {
			log.Errorln(err)
			return err
		}
		if err := o.DoAddFlow("table=0 tcp nw_dst=169.254.169.254 tp_dst=80", 10000,
			fmt.Sprintf("mod_dl_dst:%s,mod_nw_dst:%s,mod_tp_dst:%d,local",
				mac, ip, metadataPort),
			o.bridge.String()); err != nil {
			log.Errorln(err)
			return err
		}
		log.Infof("OVS: metadata server %s:%d", ip, metadataPort)

		k8sCidr := options.HostOptions.K8sClusterCidr
		if len(k8sCidr) > 0 {
			addr, mask, err := netutils2.PrefixSplit(k8sCidr)
			if err != nil {
				log.Errorln(err)
				return err
			}
			k8sCidr = fmt.Sprintf("%s/%d", addr, mask)
			log.Infof("OVS: Kubernetes cluster IP range: %s", k8sCidr)
			err = o.DoAddFlow(fmt.Sprintf("table=0 ip,nw_dst=%s", k8sCidr),
				10050, fmt.Sprintf("mod_dl_dst:%s,local", mac), o.bridge.String())
			if err != nil {
				log.Errorln(err)
				return err
			}
		}

		err := o.DoAddFlow("table=0", 0, "resubmit(,1)", o.bridge.String())
		if err != nil {
			log.Errorln(err)
			return err
		}
		err = o.DoAddFlow("table=1", 0, "normal", o.bridge.String())
		if err != nil {
			log.Errorln(err)
			return err
		}
	}
	return nil
}

func (o *SOVSBridgeDriver) ovsSetParams(params map[string]map[string]string) {
	for tbl, tblval := range params {
		for k, v := range tblval {
			procutils.NewCommand("ovs-vsctl", "set", tbl, o.bridge.String(),
				fmt.Sprintf("%s=%s", k, v)).Run()
		}
	}
}

func (o *SOVSBridgeDriver) WarmupConfig() error {
	// if options.OvsSflowBridges ...
	if options.HostOptions.EnableOpenflowController {
		// ...
	} else {
		params := map[string]map[string]string{
			"bridge": {
				"stp_enable":                           "false",
				"fail_mode":                            "standalone",
				"other-config:flow-eviction-threshold": "2500",
			},
		}
		o.ovsSetParams(params)
	}
	return nil
}

func OVSPrepare() error {
	ovs := system_service.GetService("openvswitch")
	if !ovs.IsInstalled() {
		return fmt.Errorf("Service openvswitch not installed!")
	}
	if !ovs.IsActive() {
		return ovs.Start(false)
	}
	return nil
}

func CleanOvsBridge() {
	ovsutils.CleanAllHiddenPorts()
}

func NewOVSBridgeDriver(bridge, inter, ip string) (*SOVSBridgeDriver, error) {
	base, err := NewBaseBridgeDriver(bridge, inter, ip)
	if err != nil {
		return nil, err
	}
	return &SOVSBridgeDriver{*base}, nil
}

func NewDriver(bridgeDriver, bridge, inter, ip string) (IBridgeDriver, error) {
	if bridgeDriver == "openvswitch" {
		return NewOVSBridgeDriver(bridge, inter, ip)
	} else {
		return nil, fmt.Errorf("Not Implentment")
	}
}

func Prepare(bridgeDriver string) error {
	if bridgeDriver == "openvswitch" {
		return OVSPrepare()
	} else {
		return fmt.Errorf("Not Implentment")
	}
}

func CleanDeletedPorts() {
	if options.HostOptions.BridgeDriver == "openvswitch" {
		CleanOvsBridge()
	}
}
