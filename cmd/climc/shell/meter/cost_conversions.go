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

package meter

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/meter"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.CostConversions).WithKeyword("cost-conversion")
	cmd.List(new(options.CostConversionListOptions))
	cmd.Create(new(options.CostConversionCreateOptions))
	cmd.Update(new(options.CostConversionUpdateOptions))
	cmd.Delete(new(options.CostConversionDeleteOptions))
}
