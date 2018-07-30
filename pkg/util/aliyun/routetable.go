package aliyun

import (
	"time"
)

// {"CreationTime":"2017-03-19T13:37:40Z","RouteEntrys":{"RouteEntry":[{"DestinationCidrBlock":"172.31.32.0/20","InstanceId":"","NextHopType":"local","NextHops":{"NextHop":[]},"RouteTableId":"vtb-j6c60lectdi80rk5xz43g","Status":"Available","Type":"System"},{"DestinationCidrBlock":"100.64.0.0/10","InstanceId":"","NextHopType":"service","NextHops":{"NextHop":[]},"RouteTableId":"vtb-j6c60lectdi80rk5xz43g","Status":"Available","Type":"System"}]},"RouteTableId":"vtb-j6c60lectdi80rk5xz43g","RouteTableType":"System","VRouterId":"vrt-j6c00qrol733dg36iq4qj"}

type SNextHops struct {
	NextHop []string
}

type SRouteEntry struct {
	DestinationCidrBlock string
	InstanceId           string
	NextHopType          string
	NextHops             SNextHops
	RouteTableId         string
	Status               string
	Type                 string
}

type SRouteEntrys struct {
	RouteEntry []SRouteEntry
}

type SRouteTable struct {
	CreationTime   time.Time
	RouteEntrys    SRouteEntrys
	RouteTableId   string
	RouteTableType string
	VRouterId      string
}
