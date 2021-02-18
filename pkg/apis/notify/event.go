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
	"fmt"
	"strings"
)

var (
	Event SEvent

	ActionCreate         SAction = "create"
	ActionDelete         SAction = "delete"
	ActionPendingDelete  SAction = "pending_delete"
	ActionUpdate         SAction = "update"
	ActionRebuildRoot    SAction = "rebuild_root"
	ActionResetPassword  SAction = "reset_password"
	ActionChangeConfig   SAction = "change_config"
	ActionResize         SAction = "resize"
	ActionExpiredRelease SAction = "expired_release"
	ActionExecute        SAction = "execute"
)

const (
	DelimiterInEvent = "/"
)

type SAction string

type SEvent struct {
	resourceType string
	action       SAction
}

func (se SEvent) WithResourceType(rt string) SEvent {
	se.resourceType = rt
	return se
}

func (se SEvent) WithAction(a SAction) SEvent {
	se.action = a
	return se
}

func (se SEvent) ResourceType() string {
	return se.resourceType
}

func (se SEvent) Action() SAction {
	return se.action
}

func (se SEvent) String() string {
	return strings.ToUpper(fmt.Sprintf("%s%s%s", se.ResourceType(), DelimiterInEvent, se.Action()))
}
