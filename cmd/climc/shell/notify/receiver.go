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
	cmd := shell.NewResourceCmd(&modules.NotifyReceiver).WithKeyword("notify-receiver")
	cmd.List(new(options.ReceiverListOptions))
	cmd.Create(new(options.ReceiverCreateOptions))
	cmd.Update(new(options.ReceiverUpdateOptions))
	cmd.Show(new(options.ReceiverOptions))
	cmd.Delete(new(options.ReceiverOptions))
	cmd.Perform("enable", new(options.ReceiverOptions))
	cmd.Perform("disable", new(options.ReceiverOptions))
	cmd.Perform("trigger-verify", new(options.ReceiverTriggerVerifyOptions))
	cmd.Perform("verify", new(options.ReceiverVerifyOptions))
	cmd.Perform("enable-contact-type", new(options.ReceiverEnableContactTypeInput))
	cmd.PerformClass("intellij-get", new(options.ReceiverIntellijGetOptions))
	cmd.PerformClass("get-types", new(options.ReceiverGetTypeOptions))
	cmd.Perform("get-subscription", new(options.ReceiverGetSubscriptionOptions))
	cmd.GetProperty(new(options.SReceiverRoleContactType))
}
