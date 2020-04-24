package compute

import (
	"yunion.io/x/pkg/util/netutils"
)

const (
	sVpcMappedCidr      = "100.64.0.0/17"
	VpcMappedIPMask     = 17
	sVpcMappedGatewayIP = "100.64.0.1"
	VpcMappedGatewayMac = "ee:ee:ee:ee:ee:ee"

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

var (
	vpcMappedCidr      netutils.IPV4Prefix
	vpcMappedGatewayIP netutils.IPV4Addr

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

	vpcMappedCidr = mp(netutils.NewIPV4Prefix(sVpcMappedCidr))
	vpcMappedGatewayIP = mi(netutils.NewIPV4Addr(sVpcMappedGatewayIP))

	vpcMappedHostIPStart = mi(netutils.NewIPV4Addr(sVpcMappedHostIPStart))
	vpcMappedHostIPEnd = mi(netutils.NewIPV4Addr(sVpcMappedHostIPEnd))

	vpcMappedIPStart = mi(netutils.NewIPV4Addr(sVpcMappedIPStart))
	vpcMappedIPEnd = mi(netutils.NewIPV4Addr(sVpcMappedIPEnd))
}

func VpcMappedCidr() netutils.IPV4Prefix {
	return vpcMappedCidr
}

func VpcMappedGatewayIP() netutils.IPV4Addr {
	return vpcMappedGatewayIP
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
