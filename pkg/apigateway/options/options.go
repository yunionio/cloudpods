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

package options

import (
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type GatewayOptions struct {
	DefaultRegion string `help:"Use default region while region not specific in api request"`
	CookieDomain  string `help:"specific yunionauth cookie domain" default:""`

	Timeout int `help:"Timeout in seconds, default is 300" default:"300"`

	DisableModuleApiVersion bool `help:"Disable each modules default api version" default:"false"`

	EnableTotp bool   `help:"Enable two-factor authentication" default:"false"`
	TotpIssuer string `help:"TOTP issuer" default:"Cloudpods"`

	SsoRedirectUrl     string `help:"SSO idp redirect URL"`
	SsoAuthCallbackUrl string `help:"SSO idp auth callback URL"`
	SsoLinkCallbackUrl string `help:"SSO idp link user callback URL"`
	LoginCallbackParam string `help:"Redirect callback parameter name after successful login"`

	SsoUserNotFoundCallbackUrl string `help:"failure callback URL when SSO idp link user not found"`

	ReturnFullDomainList bool `default:"true" help:"return domain list for get_regions API"`

	SessionLevelAuthCookie bool `default:"false" help:"YunionAuth cookie is valid during a browser session"`

	// 上报非敏感基础信息，帮助软件更加完善
	DisableReporting bool `default:"false" help:"Reporting data every 24 hours, report data incloud version, os, platform and usages"`

	// 启用后端服务反向代理网关
	EnableBackendServiceProxy bool `default:"false" help:"Proxy API request to backend services"`

	common_options.CommonOptions `"request_worker_count->default":"32"`

	EnableSyslogWebservice bool `help:"enable syslog webservice"`

	SyslogWebserviceUsername string `help:"syslog web service user name"`

	SyslogWebservicePassword string `help:"syslog web service password"`
}

var (
	Options *GatewayOptions
)

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*GatewayOptions)
	newOpts := newO.(*GatewayOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}

	if oldOpts.EnableBackendServiceProxy != newOpts.EnableBackendServiceProxy {
		changed = true
	}

	return changed
}
