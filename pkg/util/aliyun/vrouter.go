package aliyun

import (
	"time"
)

// "CreationTime":"2017-03-19T13:37:40Z","Description":"","RegionId":"cn-hongkong","RouteTableIds":{"RouteTableId":["vtb-j6c60lectdi80rk5xz43g"]},"VRouterId":"vrt-j6c00qrol733dg36iq4qj","VRouterName":"","VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"

type SRouteTableIds struct {
	RouteTableId []string
}

type SVRouter struct {
	CreationTime  time.Time
	Description   string
	RegionId      string
	RouteTableIds SRouteTableIds
	VRouterId     string
	VRouterName   string
	VpcId         string
}
