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

package compute

import (
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/pkg/util/netutils"
)

// IP: 20
// UDP: 8
// GENEVE HDR: 8 + 4x
// total: 36 + 4x
const VPC_OVN_ENCAP_COST = 60

const (
	VPC_EXTERNAL_ACCESS_MODE_DISTGW     = "distgw"                              // distgw only
	VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW = "eip-distgw"                          // eip when available, distgw otherwise
	VPC_EXTERNAL_ACCESS_MODE_EIP        = compute.VPC_EXTERNAL_ACCESS_MODE_EIP  // eip only
	VPC_EXTERNAL_ACCESS_MODE_NONE       = compute.VPC_EXTERNAL_ACCESS_MODE_NONE // no external access
)

var (
	VPC_EXTERNAL_ACCESS_MODES = []string{
		VPC_EXTERNAL_ACCESS_MODE_DISTGW,
		VPC_EXTERNAL_ACCESS_MODE_EIP_DISTGW,
		VPC_EXTERNAL_ACCESS_MODE_EIP,
		VPC_EXTERNAL_ACCESS_MODE_NONE,
	}
)

const (
	sVpcInterCidr    = "100.65.0.0/17"
	sVpcInterExtCidr = "100.65.0.0/30"
	VpcInterExtMask  = 30
	sVpcInterExtIP1  = "100.65.0.1"
	sVpcInterExtIP2  = "100.65.0.2"
	VpcInterExtMac1  = "ee:ee:ee:ee:ee:f0"
	VpcInterExtMac2  = "ee:ee:ee:ee:ee:f1"

	sVpcInterCidr6    = "fc00::fffe:100:65:0:0/100"
	sVpcInterExtCidr6 = "fc00::fffe:100:65:0:0/112"
	VpcInterExtMask6  = 112
	sVpcInterExtIP16  = "fc00::fffe:100:65:0:1"
	sVpcInterExtIP26  = "fc00::fffe:100:65:0:2"
)

var (
	vpcInterCidr   netutils.IPV4Prefix
	vpcInterExtIP1 netutils.IPV4Addr
	vpcInterExtIP2 netutils.IPV4Addr

	vpcInterCidr6   netutils.IPV6Prefix
	vpcInterExtIP16 netutils.IPV6Addr
	vpcInterExtIP26 netutils.IPV6Addr
)

func VpcInterCidr() netutils.IPV4Prefix {
	return vpcInterCidr
}

func VpcInterExtIP1() netutils.IPV4Addr {
	return vpcInterExtIP1
}

func VpcInterExtIP2() netutils.IPV4Addr {
	return vpcInterExtIP2
}

func VpcInterCidr6() netutils.IPV6Prefix {
	return vpcInterCidr6
}

func VpcInterExtIP16() netutils.IPV6Addr {
	return vpcInterExtIP16
}

func VpcInterExtIP26() netutils.IPV6Addr {
	return vpcInterExtIP26
}

const (
	sVpcMappedCidr      = "100.64.0.0/17"
	VpcMappedIPMask     = 17
	sVpcMappedGatewayIP = "100.64.0.1"
	VpcMappedGatewayMac = "ee:ee:ee:ee:ee:ee"

	sVpcMappedCidr6      = "fc00::fffe:100:64:0:0/100"
	VpcMappedIPMask6     = 100
	sVpcMappedGatewayIP6 = "fc00::fffe:100:64:0:1"

	// reserved: [100.64.0.2, 100.64.0.127]

	// [128, 2176 (128+2048)]
	sVpcMappedHostIPStart = "100.64.0.128"
	sVpcMappedHostIPEnd   = "100.64.8.128"

	// reserved: [10.64.8.129, 10.64.8.255]

	// [2304, 32511], 30207
	sVpcMappedIPStart = "100.64.9.0"
	sVpcMappedIPEnd   = "100.64.126.255"

	// reserved: [10.64.127.0 , 10.64.127.255]
)

const (
	sVpcEipGatewayCidr  = "100.64.128.0/17"
	VpcEipGatewayIPMask = 17
	sVpcEipGatewayIP    = "100.64.128.2"
	VpcEipGatewayMac    = "ee:ee:ee:ee:ee:ef"

	sVpcEipGatewayCidr6  = "fc00::fffe:100:64:f000:0/100"
	VpcEipGatewayIPMask6 = 100
	sVpcEipGatewayIP6    = "fc00::fffe:100:64:f000:2"

	sVpcEipGatewayIP3 = "100.64.128.3"
	VpcEipGatewayMac3 = "ee:ee:ee:ee:ee:f0"

	sVpcEipGatewayIP63 = "fc00::fffe:100:64:f000:3"
)

var (
	vpcMappedCidr      netutils.IPV4Prefix
	vpcMappedGatewayIP netutils.IPV4Addr

	vpcMappedCidr6      netutils.IPV6Prefix
	vpcMappedGatewayIP6 netutils.IPV6Addr

	vpcEipGatewayCidr netutils.IPV4Prefix
	vpcEipGatewayIP   netutils.IPV4Addr
	vpcEipGatewayIP3  netutils.IPV4Addr

	vpcEipGatewayCidr6 netutils.IPV6Prefix
	vpcEipGatewayIP6   netutils.IPV6Addr
	vpcEipGatewayIP63  netutils.IPV6Addr

	vpcMappedHostIPStart netutils.IPV4Addr
	vpcMappedHostIPEnd   netutils.IPV4Addr

	vpcMappedIPStart netutils.IPV4Addr
	vpcMappedIPEnd   netutils.IPV4Addr
)

func init() {
	mi := func(v netutils.IPV4Addr, err error) netutils.IPV4Addr {
		if err != nil {
			panic(err.Error())
		}
		return v
	}
	mp := func(v netutils.IPV4Prefix, err error) netutils.IPV4Prefix {
		if err != nil {
			panic(err.Error())
		}
		return v
	}

	mi6 := func(v netutils.IPV6Addr, err error) netutils.IPV6Addr {
		if err != nil {
			panic(err.Error())
		}
		return v
	}
	mp6 := func(v netutils.IPV6Prefix, err error) netutils.IPV6Prefix {
		if err != nil {
			panic(err.Error())
		}
		return v
	}

	vpcInterCidr = mp(netutils.NewIPV4Prefix(sVpcInterCidr))
	vpcInterExtIP1 = mi(netutils.NewIPV4Addr(sVpcInterExtIP1))
	vpcInterExtIP2 = mi(netutils.NewIPV4Addr(sVpcInterExtIP2))

	vpcInterCidr6 = mp6(netutils.NewIPV6Prefix(sVpcInterCidr6))
	vpcInterExtIP16 = mi6(netutils.NewIPV6Addr(sVpcInterExtIP16))
	vpcInterExtIP26 = mi6(netutils.NewIPV6Addr(sVpcInterExtIP26))

	vpcMappedCidr = mp(netutils.NewIPV4Prefix(sVpcMappedCidr))
	vpcMappedGatewayIP = mi(netutils.NewIPV4Addr(sVpcMappedGatewayIP))

	vpcMappedCidr6 = mp6(netutils.NewIPV6Prefix(sVpcMappedCidr6))
	vpcMappedGatewayIP6 = mi6(netutils.NewIPV6Addr(sVpcMappedGatewayIP6))

	vpcEipGatewayCidr = mp(netutils.NewIPV4Prefix(sVpcEipGatewayCidr))
	vpcEipGatewayIP = mi(netutils.NewIPV4Addr(sVpcEipGatewayIP))
	vpcEipGatewayIP3 = mi(netutils.NewIPV4Addr(sVpcEipGatewayIP3))

	vpcEipGatewayCidr6 = mp6(netutils.NewIPV6Prefix(sVpcEipGatewayCidr6))
	vpcEipGatewayIP6 = mi6(netutils.NewIPV6Addr(sVpcEipGatewayIP6))
	vpcEipGatewayIP63 = mi6(netutils.NewIPV6Addr(sVpcEipGatewayIP63))

	vpcMappedHostIPStart = mi(netutils.NewIPV4Addr(sVpcMappedHostIPStart))
	vpcMappedHostIPEnd = mi(netutils.NewIPV4Addr(sVpcMappedHostIPEnd))

	vpcMappedIPStart = mi(netutils.NewIPV4Addr(sVpcMappedIPStart))
	vpcMappedIPEnd = mi(netutils.NewIPV4Addr(sVpcMappedIPEnd))
}

func VpcMappedCidr() netutils.IPV4Prefix {
	return vpcMappedCidr
}

func VpcMappedCidr6() netutils.IPV6Prefix {
	return vpcMappedCidr6
}

func VpcMappedGatewayIP() netutils.IPV4Addr {
	return vpcMappedGatewayIP
}

func VpcMappedGatewayIP6() netutils.IPV6Addr {
	return vpcMappedGatewayIP6
}

func VpcEipGatewayCidr() netutils.IPV4Prefix {
	return vpcEipGatewayCidr
}

func VpcEipGatewayCidr6() netutils.IPV6Prefix {
	return vpcEipGatewayCidr6
}

func VpcEipGatewayIP() netutils.IPV4Addr {
	return vpcEipGatewayIP
}

func VpcEipGatewayIP6() netutils.IPV6Addr {
	return vpcEipGatewayIP6
}

func VpcEipGatewayIP3() netutils.IPV4Addr {
	return vpcEipGatewayIP3
}

func VpcEipGatewayIP63() netutils.IPV6Addr {
	return vpcEipGatewayIP63
}

func VpcMappedHostIPStart() netutils.IPV4Addr {
	return vpcMappedHostIPStart
}

func VpcMappedHostIPEnd() netutils.IPV4Addr {
	return vpcMappedHostIPEnd
}

func VpcMappedIPStart() netutils.IPV4Addr {
	return vpcMappedIPStart
}

func VpcMappedIPEnd() netutils.IPV4Addr {
	return vpcMappedIPEnd
}

func VpcOvnEncapCostStr() string {
	return fmt.Sprintf("%d", VPC_OVN_ENCAP_COST)
}

func GenVpcMappedIP6(ip string) string {
	segs := strings.Split(ip, ".")
	return fmt.Sprintf("fc00::fffe:100:64:%s:%s", segs[2], segs[3])
}
