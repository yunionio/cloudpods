package compute

import "yunion.io/x/onecloud/pkg/apis"

const (
	Modelarts_Pool_STATUS_AVAILABLE     = "available"
	Modelarts_Pool_STATUS_UNAVAILABLE   = "unavailable"
	Modelarts_Pool_STATUS_CREATING      = "creating"
	Modelarts_Pool_STATUS_DELETING      = "deleting"
	Modelarts_Pool_STATUS_DELETE_FAILED = "delete_failed"
	Modelarts_Pool_STATUS_UNKNOWN       = "unknown"
)

// 资源创建参数, 目前仅站位
type ModelartsPoolCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	// Metadata ModelartsPoolMetadata `json:"meatdata"`
	// Spec     ModelartsPoolSpec     `json:"spec"`

	ManagerId string `json:"manager_id"`
}

type ModelartsPoolMetadata struct {
}

type ModelartsPoolSpec struct {
}

// 资源返回详情
type PoolDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
}

// 资源列表请求参数
type PoolhListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	RegionalFilterListInput
	ManagedResourceListInput
}
