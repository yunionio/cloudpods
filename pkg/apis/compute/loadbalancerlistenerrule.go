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

package compute

import (
	"regexp"

	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type LoadbalancerListenerRuleDetails struct {
	apis.StatusStandaloneResourceDetails
	LoadbalancerListenerResourceInfo

	SLoadbalancerListenerRule

	BackendGroup string `json:"backend_group"`

	ProjectId string `json:"tenant_id"`
}

type LoadbalancerListenerRuleListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput
	LoadbalancerListenerFilterListInput

	// filter by backend_group
	BackendGroup string `json:"backend_group"`

	// 默认转发策略，目前只有aws用到其它云都是false
	IsDefault *bool `json:"is_default"`

	Domain []string `json:"domain"`
	Path   []string `json:"path"`
}

type LoadbalancerListenerRuleCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	// swagger: ignore
	Listener   string `json:"listener" yunion-deprecated-by:"listener_id"`
	ListenerId string `json:"listener_id"`

	BackendGroup   string `json:"backend_group" yunion-deprecated-by:"backend_group_id"`
	BackendGroupId string `json:"backend_group_id"`

	Domain               string `json:"domain"`
	Path                 string `json:"path"`
	HttpRequstRate       int    `json:"http_request_rate"`
	HttpRequstRatePerSec int    `json:"http_request_rate_per_sec"`

	Redirect       string
	RedirectCode   int64
	RedirectScheme string
	RedirectHost   string
	RedirectPath   string

	Condition string `json:"conditon"`
}

var (
	pathReg     = regexp.MustCompile(`^(?:/[a-zA-Z0-9.%$&'()*+,;=!~_-]*)*$`)
	hostPortReg = regexp.MustCompile(`(?::[0-9]{1,5})?`)
)

func (self *LoadbalancerListenerRuleCreateInput) Validate() error {
	if len(self.Domain) > 0 && !regutils.DOMAINNAME_REG.MatchString(self.Domain) {
		return httperrors.NewInputParameterError("invalid domain %s", self.Domain)
	}
	if len(self.Path) > 0 && !pathReg.MatchString(self.Path) {
		return httperrors.NewInputParameterError("invalid path %s", self.Path)
	}
	if self.HttpRequstRate < 0 {
		return httperrors.NewInputParameterError("invalid http_request_rate %d", self.HttpRequstRate)
	}
	if self.HttpRequstRatePerSec < 0 {
		return httperrors.NewInputParameterError("invalid http_request_rate_per_sec %d", self.HttpRequstRatePerSec)
	}
	if len(self.Redirect) == 0 {
		self.Redirect = LB_REDIRECT_OFF
	}
	switch self.Redirect {
	case LB_REDIRECT_OFF:
	case LB_REDIRECT_RAW:
		if self.RedirectCode < 1 {
			self.RedirectCode = LB_REDIRECT_CODE_302
		}
		if ok, _ := utils.InArray(self.RedirectCode, LB_REDIRECT_CODES); !ok {
			return httperrors.NewInputParameterError("invalid redirect_code %d", self.RedirectCode)
		}
		if len(self.RedirectPath) > 0 && !pathReg.MatchString(self.RedirectPath) {
			return httperrors.NewInputParameterError("invalid redirect_path %s", self.RedirectPath)
		}
		if len(self.RedirectScheme) > 0 && !utils.IsInStringArray(self.RedirectScheme, LB_REDIRECT_SCHEMES) {
			return httperrors.NewInputParameterError("invalid redirect_scheme %s", self.RedirectScheme)
		}
		if len(self.RedirectHost) > 0 && !hostPortReg.MatchString(self.RedirectHost) {
			return httperrors.NewInputParameterError("invalid redirect_host %s", self.RedirectHost)
		}
	default:
		return httperrors.NewInputParameterError("invalid redirect %s", self.Redirect)
	}
	return nil
}

type LoadbalancerListenerRuleUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput

	// swagger: ignore
	Listener   string `json:"listener" yunion-deprecated-by:"listener_id"`
	ListenerId string `json:"listener_id"`

	BackendGroup   string `json:"backend_group" yunion-deprecated-by:"backend_group_id"`
	BackendGroupId string `json:"backend_group_id"`

	Domain               string `json:"domain"`
	Path                 string `json:"path"`
	HttpRequstRate       int    `json:"http_request_rate"`
	HttpRequstRatePerSec int    `json:"http_request_rate_per_sec"`

	Redirect       string
	RedirectCode   int64
	RedirectScheme string
	RedirectHost   string
	RedirectPath   string

	Condition string `json:"conditon"`
}
