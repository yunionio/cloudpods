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

package k8s

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Logs   *ResourceManager
	Events *EventManager
)

type EventManager struct {
	*NamespaceResourceManager
}

func init() {
	Logs = NewResourceManager("event", "events",
		NewColumns("id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"),
		NewColumns())

	Events = &EventManager{

		NamespaceResourceManager: NewNamespaceResourceManager("k8s_event", "k8s_events",
			NewNamespaceCols("Source", "Type", "Action", "Object", "Message"),
			NewColumns()),
	}

	modules.Register(Events)
}

func (m EventManager) Get_Source(obj jsonutils.JSONObject) interface{} {
	src, _ := obj.GetString("source")
	return src
}

func (m EventManager) Get_Type(obj jsonutils.JSONObject) interface{} {
	typ, _ := obj.GetString("type")
	return typ
}

func (m EventManager) Get_Action(obj jsonutils.JSONObject) interface{} {
	act, _ := obj.Get("action")
	return act
}

func (m EventManager) Get_Object(obj jsonutils.JSONObject) interface{} {
	refObj, _ := obj.Get("involvedObject")
	if refObj == nil {
		return ""
	}
	kind, _ := refObj.GetString("kind")
	name, _ := refObj.GetString("name")
	return fmt.Sprintf("%s/%s", kind, name)
}

func (m EventManager) Get_Message(obj jsonutils.JSONObject) interface{} {
	msg, _ := obj.GetString("message")
	return msg
}
