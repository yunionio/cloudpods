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

package modules

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type LogsManager struct {
	modulebase.ResourceManager
}

var (
	Logs           LogsManager
	IdentityLogs   modulebase.ResourceManager
	ImageLogs      modulebase.ResourceManager
	ActionLogs     modulebase.ResourceManager
	CloudeventLogs modulebase.ResourceManager
	ComputeLogs    modulebase.ResourceManager
	MonitorLogs    modulebase.ResourceManager
	NotifyLogs     modulebase.ResourceManager
)

func (this *LogsManager) Get(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	service, _ := params.GetString("service")
	switch service {
	case apis.SERVICE_TYPE_LOG:
		return ActionLogs.Get(session, id, params)
	case apis.SERVICE_TYPE_IMAGE:
		return ImageLogs.Get(session, id, params)
	case apis.SERVICE_TYPE_KEYSTONE:
		return IdentityLogs.Get(session, id, params)
	case apis.SERVICE_TYPE_CLOUDEVENT:
		return CloudeventLogs.Get(session, id, params)
	case apis.SERVICE_TYPE_MONITOR:
		return MonitorLogs.Get(session, id, params)
	case apis.SERVICE_TYPE_NOTIFY:
		return NotifyLogs.Get(session, id, params)
	default:
		return ComputeLogs.Get(session, id, params)
	}
}

func (this *LogsManager) PerformClassAction(session *mcclient.ClientSession, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	service, _ := params.GetString("service")
	switch service {
	case apis.SERVICE_TYPE_LOG:
		return ActionLogs.PerformClassAction(session, action, params)
	case apis.SERVICE_TYPE_IMAGE:
		return ImageLogs.PerformClassAction(session, action, params)
	case apis.SERVICE_TYPE_KEYSTONE:
		return IdentityLogs.PerformClassAction(session, action, params)
	case apis.SERVICE_TYPE_CLOUDEVENT:
		return CloudeventLogs.PerformClassAction(session, action, params)
	case apis.SERVICE_TYPE_MONITOR:
		return MonitorLogs.PerformClassAction(session, action, params)
	case apis.SERVICE_TYPE_NOTIFY:
		return NotifyLogs.PerformClassAction(session, action, params)
	default:
		return ComputeLogs.PerformClassAction(session, action, params)
	}
}

func init() {
	IdentityLogs = NewIdentityV3Manager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"},
		[]string{})
	ImageLogs = NewImageManager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"},
		[]string{})
	ActionLogs = NewActionManager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"},
		[]string{})
	CloudeventLogs = NewCloudeventManager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"},
		[]string{})
	ComputeLogs = NewComputeManager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "user_id", "tenant", "tenant_id", "owner_tenant_id", "action", "notes"},
		[]string{})
	// ComputeLogs.SetApiVersion(mcclient.V2_API_VERSION)
	MonitorLogs = NewMonitorV2Manager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "tenant", "action", "notes"},
		[]string{})
	NotifyLogs = NewNotifyv2Manager("event", "events",
		[]string{"id", "ops_time", "obj_id", "obj_type", "obj_name", "user", "tenant", "action", "notes"},
		[]string{})

	Logs = LogsManager{ComputeLogs}
	Register(&Logs)
}
