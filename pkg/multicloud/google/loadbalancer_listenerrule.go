package google

import (
	"context"
	"fmt"
	"strings"
	"time"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerListenerRule struct {
	lbl            *SLoadbalancerListener
	pathMatcher    PathMatcher
	pathRule       PathRule
	backendService SBackendServices

	ListenerName       string `json:"listener_name"`
	BackendServiceName string `json:"backend_service_name"`
	Domain             string `json:"domain"`
	Path               string `json:"path"`
	Port               string `json:"Port"`
}

func (self *SLoadbalancerListenerRule) GetId() string {
	return fmt.Sprintf("%s::%s::%s", self.lbl.GetGlobalId(), self.Domain, strings.Join(self.pathRule.Paths, ","))
}

func (self *SLoadbalancerListenerRule) GetName() string {
	return fmt.Sprintf("%s::%s::%s", self.lbl.GetName(), self.Domain, strings.Join(self.pathRule.Paths, ","))
}

func (self *SLoadbalancerListenerRule) GetGlobalId() string {
	return self.GetId()
}

func (self *SLoadbalancerListenerRule) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancerListenerRule) Refresh() error {
	return nil
}

func (self *SLoadbalancerListenerRule) IsEmulated() bool {
	return true
}

func (self *SLoadbalancerListenerRule) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SLoadbalancerListenerRule) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadbalancerListenerRule) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SLoadbalancerListenerRule) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerListenerRule) GetProjectId() string {
	return self.lbl.GetProjectId()
}

func (self *SLoadbalancerListenerRule) GetRedirect() string {
	return ""
}

func (self *SLoadbalancerListenerRule) GetRedirectCode() int64 {
	return 0
}

func (self *SLoadbalancerListenerRule) GetRedirectScheme() string {
	return ""
}

func (self *SLoadbalancerListenerRule) GetRedirectHost() string {
	return ""
}

func (self *SLoadbalancerListenerRule) GetRedirectPath() string {
	return ""
}

func (self *SLoadbalancerListenerRule) IsDefault() bool {
	return false
}

func (self *SLoadbalancerListenerRule) GetDomain() string {
	return self.Domain
}

func (self *SLoadbalancerListenerRule) GetPath() string {
	return self.Path
}

func (self *SLoadbalancerListenerRule) GetCondition() string {
	return ""
}

func (self *SLoadbalancerListenerRule) GetBackendGroupId() string {
	return self.backendService.GetGlobalId()
}

func (self *SLoadbalancerListenerRule) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerListener) GetLoadbalancerListenerRules() ([]SLoadbalancerListenerRule, error) {
	if !self.lb.isHttpLb {
		return nil, nil
	}

	if self.rules != nil {
		return self.rules, nil
	}

	hostRules := self.lb.urlMap.HostRules
	pathMatchers := self.lb.urlMap.PathMatchers

	pmm := make(map[string]PathMatcher, 0)
	for i := range pathMatchers {
		name := pathMatchers[i].Name
		pmm[name] = pathMatchers[i]
	}

	ret := make([]SLoadbalancerListenerRule, 0)
	for _, rule := range hostRules {
		pm, ok := pmm[rule.PathMatcher]
		if !ok {
			continue
		}

		for i := range rule.Hosts {
			host := rule.Hosts[i]
			for j := range pm.PathRules {
				pr := pm.PathRules[j]

				if pr.Service != self.backendService.GetId() {
					continue
				}

				r := SLoadbalancerListenerRule{
					lbl:                self,
					backendService:     self.backendService,
					BackendServiceName: self.backendService.GetName(),
					pathMatcher:        pm,
					pathRule:           pr,

					ListenerName: self.GetName(),
					Domain:       host,
					Path:         strings.Join(pr.Paths, ","),
					Port:         self.Port,
				}

				ret = append(ret, r)
			}
		}
	}

	self.rules = ret
	return ret, nil
}
