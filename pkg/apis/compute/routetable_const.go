package compute

const (
	ROUTE_TABLE_TYPE_VPC = "VPC" // VPC路由器
	ROUTE_TABLE_TYPE_VBR = "VBR" // 边界路由器
)

const (
	ROUTE_ENTRY_TYPE_CUSTOM = "Custom" // 自定义路由
	ROUTE_ENTRY_TYPE_SYSTEM = "System" // 系统路由
)

const (
	Next_HOP_TYPE_INSTANCE        = "Instance"              // ECS实例。
	Next_HOP_TYPE_HAVIP           = "HaVip"                 // 高可用虚拟IP。
	Next_HOP_TYPE_VPN             = "VpnGateway"            // VPN网关。
	Next_HOP_TYPE_NAT             = "NatGateway"            // NAT网关。
	Next_HOP_TYPE_NETWORK         = "NetworkInterface"      // 辅助弹性网卡。
	Next_HOP_TYPE_ROUTER          = "RouterInterface"       // 路由器接口。
	Next_HOP_TYPE_IPV6            = "IPv6Gateway"           // IPv6网关。
	Next_HOP_TYPE_INTERNET        = "InternetGateway"       // Internet网关。
	Next_HOP_TYPE_EGRESS_INTERNET = "EgressInternetGateway" // egress only Internet网关。
)
