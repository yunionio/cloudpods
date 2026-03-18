package ecloud

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/util/secrules"
)

// SSecurityGroupRule 与 ecloudsdkvpc ListSecurityGroupRuleResponseContent 字段对应
type SSecurityGroupRule struct {
	multicloud.SResourceBase

	region     *SRegion
	SecgroupId string `json:"secgroupId"`

	Id             string  `json:"id"`
	Direction      string  `json:"direction"`
	Protocol       string  `json:"protocol"`
	MinPortRange   *int32  `json:"minPortRange,omitempty"`
	MaxPortRange   *int32  `json:"maxPortRange,omitempty"`
	RemoteIpPrefix string  `json:"remoteIpPrefix"`
	Description    string  `json:"description"`
	EtherType      string  `json:"etherType,omitempty"`
	AimSgid        *string `json:"aimSgid,omitempty"`
	DefaultRule    *bool   `json:"defaultRule,omitempty"`
	CreatedTime    string  `json:"createdTime,omitempty"`
	Status         *int32  `json:"status,omitempty"`
}

func (r *SSecurityGroupRule) GetGlobalId() string {
	return r.Id
}

func (r *SSecurityGroupRule) GetDirection() secrules.TSecurityRuleDirection {
	if r.Direction == "ingress" {
		return secrules.DIR_IN
	}
	return secrules.DIR_OUT
}

func (r *SSecurityGroupRule) GetPriority() int {
	return 0
}

func (r *SSecurityGroupRule) GetAction() secrules.TSecurityRuleAction {
	return secrules.SecurityRuleAllow
}

func (r *SSecurityGroupRule) GetProtocol() string {
	if r.Protocol == "" || r.Protocol == "ANY" {
		return secrules.PROTO_ANY
	}
	return strings.ToLower(r.Protocol)
}

func (r *SSecurityGroupRule) GetPorts() string {
	if r.MinPortRange != nil && r.MaxPortRange != nil {
		min, max := *r.MinPortRange, *r.MaxPortRange
		if min == max {
			return fmt.Sprintf("%d", min)
		}
		return fmt.Sprintf("%d-%d", min, max)
	}
	return ""
}

func (r *SSecurityGroupRule) GetDescription() string {
	return r.Description
}

func (r *SSecurityGroupRule) GetCIDRs() []string {
	if r.RemoteIpPrefix == "" {
		return nil
	}
	return []string{r.RemoteIpPrefix}
}

func (r *SSecurityGroupRule) Update(opts *cloudprovider.SecurityGroupRuleUpdateOptions) error {
	return errors.ErrNotSupported
}

func (r *SSecurityGroupRule) Delete() error {
	return r.region.DeleteSecurityGroupRule(r.Id)
}

