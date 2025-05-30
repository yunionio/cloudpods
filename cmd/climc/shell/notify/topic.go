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

package notify

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	options "yunion.io/x/onecloud/pkg/mcclient/options/notify"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.NotifyTopic).WithKeyword("notify-topic")
	cmd.List(new(options.TopicListOptions))
	cmd.Create(new(options.TopicCreateOptions))
	cmd.Update(new(options.TopicUpdateOptions))
	cmd.Show(new(options.TopicOptions))
	cmd.Delete(new(options.TopicOptions))
}
