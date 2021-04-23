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

package hostbridge

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/system_service"
	"yunion.io/x/onecloud/pkg/util/bwutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SOVSBridgeDriver struct {
	SBaseBridgeDriver
}

func (o *SOVSBridgeDriver) CleanupConfig() {
	//ovsutils.CleanAllHiddenPorts()
	// if enableopenflowcontroller ...
}

func (o *SOVSBridgeDriver) Exists() (bool, error) {
	data, err := procutils.NewCommand("ovs-vsctl", "list-br").Output()
	if err != nil {
		return false, errors.Wrapf(err, "failed list br: %s", data)
	}
	for _, d := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(d) == o.bridge.String() {
			return true, nil
		}
	}
	return false, nil
}

func (o *SOVSBridgeDriver) Interfaces() ([]string, error) {
	data, err := procutils.NewCommand("ovs-vsctl", "list-ifaces", o.bridge.String()).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "failed list ifaces: %s", data)
	}

	var infs = make([]string, 0)
	for _, d := range strings.Split(string(data), "\n") {
		if len(strings.TrimSpace(d)) > 0 {
			infs = append(infs, strings.TrimSpace(d))
		}
	}
	return infs, nil
}

func (o *SOVSBridgeDriver) SetupInterface() error {
	infs, err := o.Interfaces()
	if err != nil {
		return err
	}

	if o.inter != nil && !utils.IsInStringArray(o.inter.String(), infs) {
		output, err := procutils.NewCommand("ovs-vsctl", "--", "--may-exist",
			"add-port", o.bridge.String(), o.inter.String()).Output()
		if err != nil {
			return fmt.Errorf("Failed to add interface: %s, %s", err, output)
		}
	}
	return nil
}

func (o *SOVSBridgeDriver) SetupBridgeDev() error {
	exist, err := o.Exists()
	if err != nil {
		return err
	}
	if !exist {
		output, err := procutils.NewCommand("ovs-vsctl", "--", "--may-exist", "add-br", o.bridge.String()).Output()
		if err != nil {
			return errors.Wrapf(err, "ovs-vsctl add br %s failed: %s", o.bridge.String(), output)
		}
		return nil
	}
	return nil
}

func (d *SOVSBridgeDriver) PersistentMac() error {
	args := []string{
		"ovs-vsctl", "set", "Bridge", d.bridge.String(),
		"other-config:hwaddr=" + d.inter.Mac,
	}
	output, err := procutils.NewCommand(args[0], args[1:]...).Output()
	if err != nil {
		return fmt.Errorf("Ovs bridge set mac address failed %s %s", output, err)
	}
	return nil
}

func (o *SOVSBridgeDriver) GenerateIfdownScripts(scriptPath string, nic jsonutils.JSONObject, isSlave bool) error {
	return o.generateIfdownScripts(o, scriptPath, nic, isSlave)
}

func (o *SOVSBridgeDriver) GenerateIfupScripts(scriptPath string, nic jsonutils.JSONObject, isSlave bool) error {
	return o.generateIfupScripts(o, scriptPath, nic, isSlave)
}

func (o *SOVSBridgeDriver) getUpScripts(nic jsonutils.JSONObject, isSlave bool) (string, error) {
	var (
		bridge, _      = nic.GetString("bridge")
		ifname, _      = nic.GetString("ifname")
		ip, _          = nic.GetString("ip")
		mac, _         = nic.GetString("mac")
		netId, _       = nic.GetString("net_id")
		vlan, _        = nic.Int("vlan")
		vpcProvider, _ = nic.GetString("vpc", "provider")
	)

	if vpcProvider == compute.VPC_PROVIDER_OVN {
		bridge = options.HostOptions.OvnIntegrationBridge
	}

	s := "#!/bin/bash\n\n"
	s += fmt.Sprintf("SWITCH='%s'\n", bridge)
	s += fmt.Sprintf("IF='%s'\n", ifname)
	s += fmt.Sprintf("IP='%s'\n", ip)
	s += fmt.Sprintf("MAC='%s'\n", mac)
	s += fmt.Sprintf("VLAN_ID=%d\n", vlan)
	s += fmt.Sprintf("NET_ID=%s\n", netId)
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
		s += fmt.Sprintf("ip link set dev $IF mtu %d\n",
			1500+options.HostOptions.TunnelPaddingBytes)
	}
	s += "ip address flush dev $IF\n"
	s += "ip link set dev $IF up\n"
	s += "ovs-vsctl list-ifaces $SWITCH | grep -w $IF > /dev/null 2>&1\n"
	s += "if [ $? -eq '0' ]; then\n"
	s += "    ovs-vsctl del-port $SWITCH $IF\n"
	s += "fi\n"
	s += "if [ \"$VLAN_ID\" -ne \"1\" ]; then\n"
	s += "    TAG=\"tag=$VLAN_ID\"\n"
	s += "fi\n"
	s += "ovs-vsctl add-port $SWITCH $IF $TAG\n"
	if vpcProvider == compute.VPC_PROVIDER_OVN {
		if !isSlave {
			s += "ovs-vsctl set Interface $IF external_ids:iface-id=iface-$NET_ID-$IF\n"
		}
	}
	s += "PORT=$(ovs-ofctl show $SWITCH | grep -w $IF)\n"
	s += "PORT=$(echo $PORT | awk 'BEGIN{FS=\"(\"}{print $1}')\n"
	s += "OFCTL=$(ovs-vsctl get-controller $SWITCH)\n"
	s += "if [ -z \"$OFCTL\" ]; then\n"
	s += "    ovs-vsctl set Interface $IF ingress_policing_rate=$LIMIT\n"
	s += "    ovs-vsctl set Interface $IF ingress_policing_burst=$BURST\n"
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

func (o *SOVSBridgeDriver) getDownScripts(nic jsonutils.JSONObject, isSlave bool) (string, error) {
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
	s += "ip link set dev $IF down\n"
	s += "ovs-vsctl -- --if-exists del-port $SWITCH $IF\n"
	return s, nil
}

type SRule struct {
	priority int
	cond     string
	actions  string
}

func (o *SOVSBridgeDriver) RegisterHostlocalServer(mac, ip string) error {
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
	if ovs.IsEnabled() {
		err := ovs.Disable()
		if err != nil {
			log.Errorf("Disabling openvswitch service failed: %v", err)
		}
	}
	if !ovs.IsActive() {
		return ovs.Start(false)
	}
	return nil
}

func cleanOvsBridge() {
	//ovsutils.CleanAllHiddenPorts()
}

func NewOVSBridgeDriver(bridge, inter, ip string) (*SOVSBridgeDriver, error) {
	base, err := NewBaseBridgeDriver(bridge, inter, ip)
	if err != nil {
		return nil, err
	}
	ovsDrv := &SOVSBridgeDriver{*base}
	ovsDrv.drv = ovsDrv
	return ovsDrv, nil
}

func NewOVSBridgeDriverByName(bridge string) (*SOVSBridgeDriver, error) {
	return NewOVSBridgeDriver(bridge, "", "")
}
