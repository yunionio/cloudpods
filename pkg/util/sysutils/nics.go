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

package sysutils

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	sysNetPath = "/sys/class/net"
)

func Nics() ([]*types.SNicDevInfo, error) {
	if _, err := os.Stat(sysNetPath); !os.IsNotExist(err) {
		nicDevs, err := ioutil.ReadDir(sysNetPath)
		if err != nil {
			log.Errorf("ReadDir %s error: %s", sysNetPath, err)
			return nil, errors.Wrapf(err, "ioutil.ReadDir(%s)", sysNetPath)
		}
		nics := make([]*types.SNicDevInfo, 0)
		for _, nic := range nicDevs {
			netPath := filepath.Join(sysNetPath, nic.Name())
			// make sure this is a real NIC device
			if fi, err := os.Stat(filepath.Join(netPath, "device")); err != nil || fi == nil {
				continue
			} /*else if (fi.Mode() & os.ModeSymlink) == 0 {
				continue
			}*/
			nicType, err := fileutils2.FileGetContents(path.Join(netPath, "type"))
			if err != nil {
				return nil, errors.Wrap(err, "failed get nic type")
			}
			if strings.TrimSpace(nicType) == "32" {
				// include/uapi/linux/if_arp.h
				// #define ARPHRD_INFINIBAND 32		/* InfiniBand			*/
				continue // skip infiniband nic
			}

			speedStr := GetSysConfigQuiet(filepath.Join(netPath, "speed"))
			speed := 0
			if len(speedStr) > 0 {
				speed, _ = strconv.Atoi(speedStr)
			}
			carrier := GetSysConfigQuiet(filepath.Join(netPath, "carrier"))
			up := false
			if carrier == "1" {
				up = true
			}
			macStr := GetSysConfigQuiet(filepath.Join(netPath, "address"))
			permMacStr := GetSysConfigQuiet(filepath.Join(netPath, "bonding_slave/perm_hwaddr"))
			var mac net.HardwareAddr
			if len(permMacStr) > 0 {
				mac, _ = net.ParseMAC(permMacStr)
			} else if len(macStr) > 0 {
				mac, _ = net.ParseMAC(macStr)
			} else {
				// no valid mac address
				continue
			}
			mtuStr := GetSysConfigQuiet(filepath.Join(netPath, "mtu"))
			mtu := 0
			if len(mtuStr) > 0 {
				mtu, _ = strconv.Atoi(mtuStr)
			}
			nicInfo := &types.SNicDevInfo{
				Dev:   nic.Name(),
				Mac:   mac,
				Speed: speed,
				Up:    &up,
				Mtu:   mtu,
			}
			nics = append(nics, nicInfo)
		}
		return nics, nil
	}
	return nil, errors.Wrapf(httperrors.ErrNotSupported, "no such dir %s", sysNetPath)
}
