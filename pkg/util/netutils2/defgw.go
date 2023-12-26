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

package netutils2

import (
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
)

type SNicInfo struct {
	IpAddr  string
	Gateway string
	MacAddr string
}

type SNicInfoList []SNicInfo

func (nics SNicInfoList) Add(ip, mac, gw string) SNicInfoList {
	return append(nics, SNicInfo{
		IpAddr:  ip,
		MacAddr: mac,
		Gateway: gw,
	})
}

func (nics SNicInfoList) FindDefaultNicMac() (string, int) {
	var intMac, exitMac string
	var intIdx, exitIdx int
	for i, nic := range nics {
		if len(nic.IpAddr) == 0 {
			continue
		}
		if len(nic.Gateway) == 0 {
			continue
		}
		addr, err := netutils.NewIPV4Addr(nic.IpAddr)
		if err != nil {
			log.Errorf("NewIPV4Addr %s fail %s", nic.IpAddr, err)
			continue
		}
		if len(exitMac) == 0 || len(intMac) == 0 {
			isExit := netutils.IsExitAddress(addr)
			if len(exitMac) == 0 && isExit {
				exitMac = nic.MacAddr
				exitIdx = i
			}
			if len(intMac) == 0 && !isExit {
				intMac = nic.MacAddr
				intIdx = i
			}
		} else if len(exitMac) > 0 && len(intMac) > 0 {
			break
		}
	}
	if len(exitMac) > 0 {
		return exitMac, exitIdx
	}
	if len(intMac) > 0 {
		return intMac, intIdx
	}
	return "", -1
}
