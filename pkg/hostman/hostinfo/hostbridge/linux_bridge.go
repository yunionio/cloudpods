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
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/iproute2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func NewLinuxBridgeDeriver(bridge, inter, ip string) (*SLinuxBridgeDriver, error) {
	base, err := NewBaseBridgeDriver(bridge, inter, ip)
	if err != nil {
		return nil, err
	}
	linuxBridgeDrv := &SLinuxBridgeDriver{*base}
	linuxBridgeDrv.drv = linuxBridgeDrv
	return linuxBridgeDrv, nil
}

func LinuxBridgePrepare() error {
	return nil
}

func cleanLinuxBridge() {
	// pass
}

type SLinuxBridgeDriver struct {
	SBaseBridgeDriver
}

func (l *SLinuxBridgeDriver) Exists() (bool, error) {
	data, err := procutils.NewCommand("brctl", "show").Output()
	if err != nil {
		return false, err
	}

	re := regexp.MustCompile(`\s+`)
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		info := re.Split(string(line), -1)
		if info[0] == l.bridge.String() {
			return true, nil
		}
	}
	return false, nil
}

func (l *SLinuxBridgeDriver) Interfaces() ([]string, error) {
	data, err := procutils.NewCommand("brctl", "show", l.bridge.String()).Output()
	if err != nil {
		return nil, err
	}

	infs := make([]string, 0)
	re := regexp.MustCompile(`\s+`)
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		info := re.Split(string(line), -1)
		infs = append(infs, info[len(info)-1])
	}
	return infs, nil
}

func (l *SLinuxBridgeDriver) GenerateIfdownScripts(scriptPath string, nic jsonutils.JSONObject, isSlave bool) error {
	return l.generateIfdownScripts(l, scriptPath, nic, isSlave)
}

func (l *SLinuxBridgeDriver) GenerateIfupScripts(scriptPath string, nic jsonutils.JSONObject, isSlave bool) error {
	return l.generateIfupScripts(l, scriptPath, nic, isSlave)
}

func (l *SLinuxBridgeDriver) getUpScripts(nic jsonutils.JSONObject, isSlave bool) (string, error) {
	s := "#!/bin/bash\n\n"
	s += fmt.Sprintf("switch='%s'\n", l.bridge)
	if options.HostOptions.TunnelPaddingBytes > 0 {
		s += fmt.Sprintf("ip link set dev $1 mtu %d\n", 1500+options.HostOptions.TunnelPaddingBytes)
	}
	s += "ip address flush dev $1\n"
	s += "ip link set dev $1 up\n"
	s += "brctl addif ${switch} $1\n"
	return s, nil
}

func (l *SLinuxBridgeDriver) getDownScripts(nic jsonutils.JSONObject, isSlave bool) (string, error) {
	s := "#!/bin/sh\n\n"
	s += fmt.Sprintf("switch='%s'\n", l.bridge)
	s += "brctl show ${switch} | grep $1\n"
	s += "if [ $? -ne '0' ]; then\n"
	s += "    exit 0\n"
	s += "fi\n"
	s += "ip addr flush dev $1\n"
	s += "ip link set dev $1 down\n"
	s += "brctl delif ${switch} $1\n"
	return s, nil
}

func (l *SLinuxBridgeDriver) SetupBridgeDev() error {
	exist, err := l.Exists()
	if err != nil {
		return err
	}
	if !exist {
		output, err := procutils.NewCommand("brctl", "addbr", l.bridge.String()).Output()
		if err != nil {
			return errors.Wrapf(err, "Failed to create bridge %s", output)
		}
	}
	return nil
}

func (d *SLinuxBridgeDriver) PersistentMac() error {
	l := iproute2.NewLink(d.bridge.String()).Address(d.inter.Mac)
	if err := l.Err(); err != nil {
		return fmt.Errorf("Linux bridge set mac address failed: %v", err)
	}
	return nil
}

func (l *SLinuxBridgeDriver) RegisterHostlocalServer(mac, ip string) error {
	metadataPort := l.GetMetadataServerPort()
	metadataServerLoc := fmt.Sprintf("%s:%d", ip, metadataPort)
	hostDnsServerLoc := fmt.Sprintf("%s:%d", ip, 53)

	// cmd := "iptables -t nat -F"
	// cmd1 := strings.Split(cmd, " ")
	// output, err := procutils.NewCommand(cmd1[0], cmd1[1:]...).Output()
	// if err != nil {
	// 	log.Errorf("Clean iptables failed: %s", output)
	// 	return err
	// }

	cmd := "iptables -t nat -A PREROUTING -s 0.0.0.0/0"
	cmd += " -d 169.254.169.254/32 -p tcp -m tcp --dport 80"
	cmd += fmt.Sprintf(" -j DNAT --to-destination %s", metadataServerLoc)
	cmd1 := strings.Split(cmd, " ")
	output, err := procutils.NewCommand(cmd1[0], cmd1[1:]...).Output()
	if err != nil {
		log.Errorf("Inject DNAT rule failed: %s", output)
		return err
	}

	cmd = "sysctl -w net.ipv4.ip_forward=1"
	cmd1 = strings.Split(cmd, " ")
	output, err = procutils.NewCommand(cmd1[0], cmd1[1:]...).Output()
	if err != nil {
		log.Errorf("Enable ip forwarding failed: %s", output)
		return err
	}

	log.Infof("Bridge: metadata server=%s", metadataServerLoc)
	log.Infof("Bridge: host dns server=%s", hostDnsServerLoc)
	return nil
}

func (l *SLinuxBridgeDriver) SetupInterface() error {
	infs, err := l.Interfaces()
	if err != nil {
		return err
	}
	if l.inter != nil && !utils.IsInStringArray(l.inter.String(), infs) {
		err := procutils.NewCommand("brctl", "addif", l.bridge.String(), l.inter.String()).Run()
		if err != nil {
			return fmt.Errorf("Failed to add interface %s", l.inter)
		}
	}
	return nil
}
