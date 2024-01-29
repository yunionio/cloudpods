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

package cloudproxy

import (
	"yunion.io/x/onecloud/pkg/util/choices"
)

const (
	SERVICE_TYPE    = "cloudproxy"
	SERVICE_VERSION = ""
)

const (
	PM_SCOPE_VPC     = "vpc"
	PM_SCOPE_NETWORK = "network"
)

var PM_SCOPES = choices.NewChoices(
	PM_SCOPE_VPC,
	PM_SCOPE_NETWORK,
)

const (
	FORWARD_TYPE_LOCAL  = "local"
	FORWARD_TYPE_REMOTE = "remote"
)

var FORWARD_TYPES = choices.NewChoices(
	FORWARD_TYPE_LOCAL,
	FORWARD_TYPE_REMOTE,
)

const (
	BindPortMin = 20000
	BindPortMax = 60000
)
