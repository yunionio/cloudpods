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
	"yunion.io/x/onecloud/pkg/i18n"
)

const (
	CLOUD_PROVIDER_ONECLOUD_EN = "YunionCloud"
	CLOUD_PROVIDER_ONECLOUD_CN = "云联壹云"
)

var ComputeI18nTable = i18n.Table{}

func init() {
	ComputeI18nTable.Set(CLOUD_PROVIDER_ONECLOUD, i18n.NewTableEntry().
		EN(CLOUD_PROVIDER_ONECLOUD_EN).
		CN(CLOUD_PROVIDER_ONECLOUD_CN),
	)
}
