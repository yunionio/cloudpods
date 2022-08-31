package compute

import (
	"yunion.io/x/onecloud/pkg/apis"
)

const (
	MODELARTS_POOL_STATUS_RUNNING  = "Running"
	Modelarts_Pool_STATUS_ABNORMAL = "Abnormal"
	Modelarts_Pool_STATUS_CREATING = "Creating"
	Modelarts_Pool_STATUS_DELETING = "Deleting"
	Modelarts_Pool_STATUS_ERROR    = "Error"
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
type ModelartsPoolDetails struct {
	apis.SVirtualResourceBase
	apis.SExternalizedResourceBase
	SBillingResourceBase
	ManagedResourceInfo
}

// 资源列表请求参数
type PoolhListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	apis.DeletePreventableResourceBaseListInput

	RegionalFilterListInput
	ManagedResourceListInput
}

func (self ModelartsPoolDetails) GetMetricTags() map[string]string {
	ret := map[string]string{
		"modelarts_pool_id":   self.Id,
		"modelarts_pool_name": self.Name,
		"status":              self.Status,
		"tenant_id":           self.ProjectId,
		"brand":               self.Brand,
		"domain_id":           self.DomainId,
		"account_id":          self.AccountId,
		"account":             self.Account,
	}
	return ret
}
