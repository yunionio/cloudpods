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

	Timeout int `help:"Timeout in seconds, default is 300" default:"300"`

	DisableModuleApiVersion bool `help:"Disable each modules default api version" default:"false"`

	EnableTotp bool `help:"Enable two-factor authentication"  default:"false"`

	SqlitePath string `help:"sqlite db path" default:"/etc/yunion/data/yunionapi.db"`

	common_options.CommonOptions
}

var (
	Options GatewayOptions
)
