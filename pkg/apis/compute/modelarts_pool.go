package compute

import (
	"yunion.io/x/onecloud/pkg/apis"
)

const (
	MODELARTS_POOL_STATUS_RUNNING  = "running"
	MODELARTS_POOL_STATUS_ABNORMAL = "abnormal"
	MODELARTS_POOL_STATUS_CREATING = "creating"
	MODELARTS_POOL_STATUS_DELETING = "deleting"
	MODELARTS_POOL_STATUS_ERROR    = "error"
)

type ModelartsPoolCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput
	CloudproviderResourceInput
}

type ModelartsPoolUpdateInput struct {
	apis.StatusInfrasResourceBaseCreateInput
	CloudproviderResourceInput
	WorkType string `json:"work_type"`
}

// 资源返回详情
type ModelartsPoolDetails struct {
	apis.SVirtualResourceBase
	apis.VirtualResourceDetails

	apis.SExternalizedResourceBase
	SBillingResourceBase
	ManagedResourceInfo
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

type ModelartsPoolListInput struct {
	apis.VirtualResourceListInput
	apis.ExternalizedResourceBaseListInput
	ManagedResourceListInput
	apis.DeletePreventableResourceBaseListInput
}
