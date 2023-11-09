package compute

import "yunion.io/x/onecloud/pkg/apis"

// 资源创建参数, 目前仅占位
type SSLCertificateCreateInput struct {
}

// 资源返回详情
type SSLCertificateDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
}

// 资源列表请求参数
type SSLCertificateListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	RegionalFilterListInput
	ManagedResourceListInput
}
