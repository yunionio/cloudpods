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
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Containers)
	cmd.Create(new(options.ContainerCreateOptions))
	cmd.List(new(options.ContainerListOptions))
	cmd.BatchDelete(new(options.ContainerDeleteOptions))
	cmd.BatchPerform("stop", new(options.ContainerStopOptions))
	cmd.BatchPerform("start", new(options.ContainerStartOptions))
}
