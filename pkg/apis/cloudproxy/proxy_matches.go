package cloudproxy

import "yunion.io/x/onecloud/pkg/apis"

type ProxyMatchListInput struct {
	apis.VirtualResourceListInput

	// 代理节点（ID或Name）
	ProxyEndpointId string `json:"proxy_endpoint_id"`
	// swagger:ignore
	// Deprecated
	// Filter by proxy endpoint Id
	ProxyEndpoint string `json:"proxy_endpoint" yunion-deprecated-by:"proxy_endpoint_id"`
}
