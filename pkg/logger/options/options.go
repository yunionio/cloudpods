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
	"yunion.io/x/onecloud/pkg/logger/extern"
)

type SLoggerOptions struct {
	common_options.CommonOptions

	common_options.DBOptions

	SyslogUrl string `help:"external syslog url, e.g. tcp://localhost:1234@cloud"`

	EnableSeparateAdminLog bool `help:"enable separate log for auditor admin" default:"false"`

	SecadminRoleNames                  []string `help:"role names of security admin" default:"sys_secadmin,domain_secadmin"`
	OpsadminRoleNames                  []string `help:"role names of operation admin" default:"sys_opsadmin,domain_opsadmin"`
	AuditorRoleNames                   []string `help:"role names of auditor admin" default:"sys_adtadmin,domain_adtadmin"`
	ActionLogExceedCount               int      `help:"trigger notification when action log exceed count" default:"-1"`
	ActionLogExceedCountNotifyInterval string   `help:"trigger notification interval" default:"5m"`
}

var (
	Options SLoggerOptions
)

func OnOptionsChange(oldOptions, newOptions interface{}) bool {
	oldOpts := oldOptions.(*SLoggerOptions)
	newOpts := newOptions.(*SLoggerOptions)

	changed := false
	if common_options.OnBaseOptionsChange(&oldOpts.BaseOptions, &newOpts.BaseOptions) {
		changed = true
	}

	if common_options.OnDBOptionsChange(&oldOpts.DBOptions, &newOpts.DBOptions) {
		changed = true
	}

	if oldOpts.SyslogUrl != newOpts.SyslogUrl {
		err := extern.InitSyslog(newOpts.SyslogUrl)
		if err != nil {
			// reset syslog writer error, restart the service to take effect
			changed = true
		}
	}

	return changed
}
