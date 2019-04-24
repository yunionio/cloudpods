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

package bwutils

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/netutils"
)

func GetBwValue(nicDesc jsonutils.JSONObject) int {
	bw, err := nicDesc.Int("bw")
	if err != nil {
		ip, err := nicDesc.GetString("ip")
		if err != nil {
			bw = 1
		} else {
			ipv4, err := netutils.NewIPV4Addr(ip)
			if err == nil && netutils.IsExitAddress(ipv4) {
				bw = 1
			} else {
				bw = 10000
			}
		}
	}
	return int(bw)
}

func GetDownloadBwValue(nicDesc jsonutils.JSONObject, bwDownloadBandwidth int) (int, error) {
	ip, _ := nicDesc.GetString("ip")
	ifname, _ := nicDesc.GetString("ifname")
	if len(ip) > 0 {
		ipv4, err := netutils.NewIPV4Addr(ip)
		if err != nil {
			return 0, err
		}
		if netutils.IsExitAddress(ipv4) && len(ifname) > 0 && bwDownloadBandwidth > 0 {
			bw := GetBwValue(nicDesc)
			if bw > bwDownloadBandwidth {
				return bw, nil
			} else {
				return bwDownloadBandwidth, nil
			}
		}
	}
	return 0, nil
}

func GetOvsBwValues(nicDesc jsonutils.JSONObject) (int, int, error) {
	var bwOvs int
	bw := GetBwValue(nicDesc)
	ip, err := nicDesc.GetString("ip")
	if err == nil {
		ipv4, err := netutils.NewIPV4Addr(ip)
		if err != nil {
			return 0, 0, err
		}
		if netutils.IsExitAddress(ipv4) {
			bwOvs = 1000
			if bwOvs > bw*15 {
				bwOvs = bw * 15
			}
		} else {
			bwOvs = bw
		}
	} else {
		bwOvs = bw
	}
	return bwOvs * 1000, bwOvs * 2000, nil
}
