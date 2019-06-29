package huawei

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SElbListenerPolicy struct {
	region   *SRegion
	lb       *SLoadbalancer
	listener *SElbListener

	RedirectPoolID     string  `json:"redirect_pool_id"`
	RedirectListenerID *string `json:"redirect_listener_id"`
	Description        string  `json:"description"`
	AdminStateUp       bool    `json:"admin_state_up"`
	Rules              []Rule  `json:"rules"`
	TenantID           string  `json:"tenant_id"`
	ProjectID          string  `json:"project_id"`
	ListenerID         string  `json:"listener_id"`
	RedirectURL        *string `json:"redirect_url"`
	ProvisioningStatus string  `json:"provisioning_status"`
	Action             string  `json:"action"`
	Position           int64   `json:"position"`
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
}

type Rule struct {
	ID string `json:"id"`
}

type SElbListenerPolicyRule struct {
	region *SRegion
	policy *SElbListenerPolicy

	CompareType        string      `json:"compare_type"`
	ProvisioningStatus string      `json:"provisioning_status"`
	AdminStateUp       bool        `json:"admin_state_up"`
	TenantID           string      `json:"tenant_id"`
	ProjectID          string      `json:"project_id"`
	Invert             bool        `json:"invert"`
	Value              string      `json:"value"`
	Key                interface{} `json:"key"`
	Type               string      `json:"type"`
	ID                 string      `json:"id"`
}

func (self *SElbListenerPolicy) GetId() string {
	return self.ID
}

func (self *SElbListenerPolicy) GetName() string {
	return self.Name
}

func (self *SElbListenerPolicy) GetGlobalId() string {
	return self.GetId()
}

// 负载均衡没有启用禁用操作
func (self *SElbListenerPolicy) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbListenerPolicy) Refresh() error {
	ret := &SElbListenerPolicy{}
	err := DoGet(self.lb.region.ecsClient.ElbL7policies.Get, self.GetId(), nil, ret)
	if err != nil {
		return err
	}

	err = jsonutils.Update(self, ret)
	if err != nil {
		return err
	}

	return nil
}

func (self *SElbListenerPolicy) IsEmulated() bool {
	return false
}

func (self *SElbListenerPolicy) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SElbListenerPolicy) GetProjectId() string {
	return ""
}

func (self *SElbListenerPolicy) GetRules() ([]SElbListenerPolicyRule, error) {
	ret, err := self.region.GetLoadBalancerPolicyRules(self.GetId())
	if err != nil {
		return nil, err
	}

	for i := range ret {
		ret[i].policy = self
	}

	return ret, nil
}

func (self *SElbListenerPolicy) GetDomain() string {
	rules, err := self.GetRules()
	if err != nil {
		log.Errorf("loadbalancer rule GetDomain %s", err)
	}

	for i := range rules {
		if rules[i].Type == "HOST_NAME" {
			return rules[i].Value
		}
	}

	return ""
}

func (self *SElbListenerPolicy) GetPath() string {
	rules, err := self.GetRules()
	if err != nil {
		log.Errorf("loadbalancer rule GetPath %s", err)
	}

	for i := range rules {
		if rules[i].Type == "PATH" {
			return rules[i].Value
		}
	}

	return ""
}

func (self *SElbListenerPolicy) GetBackendGroupId() string {
	return self.RedirectPoolID
}

func (self *SElbListenerPolicy) Delete() error {
	return self.region.DeleteLoadBalancerPolicy(self.GetId())
}

func (self *SRegion) DeleteLoadBalancerPolicy(policyId string) error {
	return DoDelete(self.ecsClient.ElbL7policies.Delete, policyId, nil, nil)
}
