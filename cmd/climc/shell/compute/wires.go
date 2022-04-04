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
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Wires).WithKeyword("wire")
	cmd.List(new(options.WireListOptions))
	cmd.Create(new(options.WireCreateOptions))
	cmd.Update(new(options.WireUpdateOptions))
	cmd.Show(new(options.WireOptions))
	cmd.Delete(new(options.WireOptions))
	cmd.Perform("public", new(options.WirePublicOptions))
	cmd.Perform("private", new(options.WireOptions))
	cmd.Perform("change-owner-candidate-domains", new(options.WireOptions))
	cmd.PerformWithKeyword("merge", "merge-from", new(options.WireMergeOptions))
	cmd.Perform("merge-network", new(options.WireOptions))
	cmd.Get("topology", &options.WireOptions{})
	cmd.Perform("set-class-metadata", &options.ResourceMetadataOptions{})
}
