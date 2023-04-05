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

package hostinfo

import (
	"fmt"
	"os"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/system_service"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/ovsutils"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	ErrOvnService = errors.Error("ovn controller")
	ErrOvnConfig  = errors.Error("ovn controller configuration")
)

type OvnHelper struct {
	hi *SHostInfo
}

func NewOvnHelper(hi *SHostInfo) *OvnHelper {
	oh := &OvnHelper{
		hi: hi,
	}
	return oh
}

func (oh *OvnHelper) Init() (err error) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			err = panicVal.(error)
		}
	}()
	oh.mustPrepOvsdbConfig()
	oh.configBridgeMtu()
	if _, ok := ovnContainerImageTag(); !ok {
		oh.mustPrepService()
	}
	return nil
}

func (oh *OvnHelper) configBridgeMtu() {
	timer := time.NewTimer(time.Minute)
	go func() {
		<-timer.C
		err := oh.ensureConfigBridgeMtu()
		if err != nil {
			log.Errorf("configuring mtu fail: %s, retry...", err)
			oh.configBridgeMtu()
		} else {
			opts := &options.HostOptions
			log.Infof("set brvpc MTU to %d success!", opts.OvnUnderlayMtu)
		}
	}()
}

func (oh *OvnHelper) ensureConfigBridgeMtu() error {
	opts := &options.HostOptions
	args := []string{"set", "Interface", opts.OvnIntegrationBridge, fmt.Sprintf("mtu_request=%d", opts.OvnUnderlayMtu)}
	output, err := procutils.NewCommand("ovs-vsctl", args...).Output()
	if err != nil {
		return errors.Wrapf(err, "ovs-vsctl %s", string(output))
	}
	return nil
}

func (oh *OvnHelper) mustPrepOvsdbConfig() {
	var (
		args = []string{"set", "Open_vSwitch", "."}
		opts = &options.HostOptions
	)
	{
		if opts.OvnIntegrationBridge == "" {
			panic(errors.Wrap(ErrOvnConfig, "bad config: ovn_integration_bridge"))
		}
		args = append(args, fmt.Sprintf("external_ids:ovn-bridge=%s",
			opts.OvnIntegrationBridge))
	}
	{
		encapIp := opts.OvnEncapIp
		if encapIp == "" {
			var (
				err  error
				meth = opts.OvnEncapIpDetectionMethod
			)
			switch {
			case strings.HasPrefix(meth, "can-reach:"):
				encapIp, err = netutils2.MyIPTo(meth[10:])
			default:
				encapIp, err = netutils2.MyIP()
			}
			if err != nil {
				panic(errors.Wrapf(ErrOvnConfig, "determine encap ip, method: %q: %v", meth, err))
			}
		}
		args = append(args, "external_ids:ovn-encap-type=geneve")
		args = append(args, fmt.Sprintf("external_ids:ovn-encap-ip=%s", encapIp))
	}
	{
		if opts.OvnSouthDatabase == "" {
			panic(errors.Wrap(ErrOvnConfig, "bad config: ovn_south_database"))
		}
		db, err := ovsutils.NormalizeDbHost(opts.OvnSouthDatabase)
		if err != nil {
			panic(errors.Wrap(err, "normalize db host"))
		}
		opts.OvnSouthDatabase = db
		args = append(args, fmt.Sprintf("external_ids:ovn-remote=%s",
			opts.OvnSouthDatabase))
	}
	log.Debugf("exec %s", strings.Join(args, " "))
	output, err := procutils.NewCommand("ovs-vsctl", args...).Output()
	if err != nil {
		panic(errors.Wrapf(err, "configuring ovn-controller: %s", string(output)))
	}
}

func (oh *OvnHelper) mustPrepService() {
	ovn := system_service.GetService("ovn-controller")
	if !ovn.IsInstalled() {
		panic(errors.Wrap(ErrOvnService, "not installed"))
	}
	if ovn.IsEnabled() {
		// - ovn-controller Requires "openvswitch.service"
		// - openvswitch service should be disabled on startup
		if err := ovn.Disable(); err != nil {
			panic(errors.Wrap(err, "disable ovn-controller on startup"))
		}
	}
	if err := ovn.Start(false); err != nil {
		panic(errors.Wrap(err, "start ovn-controller"))
	}
}

func MustGetOvnVersion() string {
	if tag, _ := ovnContainerImageTag(); tag != "" {
		return tag
	}
	output, err := procutils.NewRemoteCommandAsFarAsPossible("ovn-controller", "--version").Output()
	if err != nil {
		return ""
	}
	return ovnExtractVersion(string(output))
}

func HasOvnSupport() bool {
	if OvnControllerInsideContainer() {
		return true
	}
	ver := MustGetOvnVersion()
	if ver != "" {
		return true
	}
	return false
}

func OvnControllerInsideContainer() bool {
	tag, _ := ovnContainerImageTag()
	if tag != "" {
		return true
	}
	return false
}

func ovnContainerImageTag() (string, bool) {
	return os.LookupEnv("OVN_CONTAINER_IMAGE_TAG")
}

func ovnExtractVersion(in string) string {
	r := make([]rune, 0, 8)
	var (
		dot   = false
		ndot  = 0
		digit = 0
	)
	reset := func() {
		dot = false
		ndot = 0
		digit = 0
	}
	for _, c := range in {
		switch {
		case c == '.':
			if dot || digit == 0 {
				reset()
				continue
			}
			r = append(r, c)
			dot = true
			ndot += 1
			digit = 0
		case c >= '0' && c <= '9':
			dot = false
			if digit < 3 {
				r = append(r, c)
				digit += 1
				continue
			}
			reset()
		default:
			if ndot > 0 && ndot < 3 {
				return string(r)
			}
			reset()
		}
	}
	if ndot > 0 && ndot < 3 {
		return string(r)
	}
	return ""
}
