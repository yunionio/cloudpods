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

package ansible

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/ansible"
	options "yunion.io/x/onecloud/pkg/mcclient/options/ansible"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.AnsiblePlaybookReference).WithKeyword("ansibleplaybook-reference")
	cmd.List(new(options.APRListOptions))
	cmd.Show(new(options.APROptions))
	cmd.Perform("run", new(options.APRRunOptions))
	cmd.Perform("stop", new(options.APRStopOptions))

	cmd1 := shell.NewResourceCmd(&modules.AnsiblePlaybookInstance).WithKeyword("ansibleplaybook-instance")
	cmd1.List(new(options.APIListOptions))
	cmd1.Show(new(options.APIOptions))
	cmd1.Perform("run", new(options.APIOptions))
}
