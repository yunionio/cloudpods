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
	"path/filepath"
	"strconv"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/httperrors"
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
			if _, err := os.Stat(filepath.Join(netPath, "device")); os.IsNotExist(err) {
				continue
			}
			speedStr := GetSysConfig(filepath.Join(netPath, "speed"))
			speed := 0
			if len(speedStr) > 0 {
				speed, _ = strconv.Atoi(speedStr)
			}
			carrier := GetSysConfig(filepath.Join(netPath, "carrier"))
			up := false
			if carrier == "1" {
				up = true
			}
			mac, _ := net.ParseMAC(GetSysConfig(filepath.Join(netPath, "address")))
			mtuStr := GetSysConfig(filepath.Join(netPath, "mtu"))
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
