// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SLoadbalancerListenerRule struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase

	listener *SLoadBalancerListener
	lbbg     *SLoadbalancerBackendGroup
	redirect *RedirectConfiguration
	// urlPathMaps -> pathRules
	Name       string             `json:"name"`
	ID         string             `json:"id"`
	Domain     string             `json:"domain"`
	Properties PathRuleProperties `json:"properties"`
}

func (self *SLoadbalancerListenerRule) GetId() string {
	return self.ID
}

func (self *SLoadbalancerListenerRule) GetName() string {
	return self.Name
}

func (self *SLoadbalancerListenerRule) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadbalancerListenerRule) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "Succeeded":
		return api.LB_STATUS_ENABLED
	case "Failed":
		return api.LB_STATUS_DISABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadbalancerListenerRule) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadbalancerListenerRule) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SLoadbalancerListenerRule) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

func (self *SLoadbalancerListenerRule) GetProjectId() string {
	return getResourceGroup(self.GetId())
}

func (self *SLoadbalancerListenerRule) IsDefault() bool {
	return false
}

func (self *SLoadbalancerListenerRule) GetDomain() string {
	return self.Domain
}

func (self *SLoadbalancerListenerRule) GetPath() string {
	if len(self.Properties.Paths) > 0 {
		return self.Properties.Paths[0]
	}

	return ""
}

func (self *SLoadbalancerListenerRule) GetCondition() string {
	return ""
}

func (self *SLoadbalancerListenerRule) GetBackendGroupId() string {
	if self.lbbg != nil {
		return self.lbbg.GetId()
	}

	return ""
}

func (self *SLoadbalancerListenerRule) Delete(ctx context.Context) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadbalancerListenerRule) GetRedirect() string {
	if self.redirect != nil {
		return api.LB_REDIRECT_RAW
	}

	return api.LB_REDIRECT_OFF
}

func (self *SLoadbalancerListenerRule) GetRedirectCode() int64 {
	if self.redirect == nil {
		return 0
	}

	switch self.redirect.Properties.RedirectType {
	case "Permanent":
		return api.LB_REDIRECT_CODE_301
	case "Found":
		return api.LB_REDIRECT_CODE_302
	case "Temporary", "SeeOther":
		return api.LB_REDIRECT_CODE_307
	default:
		return 0
	}
}

func (self *SLoadbalancerListenerRule) getRedirectUrl() *url.URL {
	if self.redirect == nil {
		return nil
	}

	if len(self.redirect.Properties.TargetUrl) == 0 {
		return nil
	}

	_url := self.redirect.Properties.TargetUrl
	if matched, _ := regexp.MatchString("^\\w{0,5}://", _url); !matched {
		_url = "http://" + _url
	}

	u, err := url.Parse(_url)
	if err != nil {
		log.Debugf("url Parse %s : %s", self.redirect.Properties.TargetUrl, err)
		return nil
	}

	return u
}
func (self *SLoadbalancerListenerRule) GetRedirectScheme() string {
	u := self.getRedirectUrl()
	if u == nil {
		return ""
	}

	return strings.ToLower(u.Scheme)
}

func (self *SLoadbalancerListenerRule) GetRedirectHost() string {
	u := self.getRedirectUrl()
	if u == nil {
		if self.redirect != nil && len(self.redirect.Properties.TargetListener.ID) > 0 {
			segs := strings.Split(self.redirect.Properties.TargetListener.ID, "/")
			return segs[len(segs)-1]
		}
		return ""
	}

	return u.Host
}

func (self *SLoadbalancerListenerRule) GetRedirectPath() string {
	u := self.getRedirectUrl()
	if u == nil {
		return ""
	}

	return u.Path
}
