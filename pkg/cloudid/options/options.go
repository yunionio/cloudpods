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

type SCloudIdOptions struct {
	common_options.CommonOptions
	common_options.DBOptions

	CloudaccountSyncIntervalMinutes  int `help:"frequency to sync region cloudaccount task" default:"3"`
	SAMLProviderSyncIntervalHours    int `help:"frequency to sync account saml provider task" default:"3"`
	SystemPoliciesSyncIntervalHours  int `help:"frequency to sync region cloudaccount task" default:"24"`
	CloudIdResourceSyncIntervalHours int `help:"frequency to sync region cloudpolicy task" default:"3"`
	CloudroleSyncIntervalHours       int `help:"frequency to sync region cloudroles task" default:"12"`

	CloudSAMLMetadataPath string `help:"path to store SAML sp metadata file of cloud providers" default:"/opt/yunion/share/saml/sp-metadata"`
}

var (
	Options SCloudIdOptions
)

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*SCloudIdOptions)
	newOpts := newO.(*SCloudIdOptions)

	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}
	return changed
}
