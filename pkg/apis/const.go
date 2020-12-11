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

package apis

const (
	SERVICE_TYPE_IMAGE             = "image"
	SERVICE_TYPE_OFFLINE_CLOUDMETA = "offlinecloudmeta"
	SERVICE_TYPE_CLOUDID           = "cloudid"
	SERVICE_TYPE_CLOUDEVENT        = "cloudevent"
	SERVICE_TYPE_DEVTOOL           = "devtool"
	SERVICE_TYPE_ANSIBLE           = "ansible"
	SERVICE_TYPE_CLOUDMETA         = "cloudmeta"
	SERVICE_TYPE_WEBSOCKET         = "websocket"
	SERVICE_TYPE_AUTOUPDATE        = "autoupdate"
	SERVICE_TYPE_YUNIONCONF        = "yunionconf"
	SERVICE_TYPE_YUNIONAGENT       = "yunionagent"
	SERVICE_TYPE_METER             = "meter"
	SERVICE_TYPE_SCHEDULER         = "scheduler"
	SERVICE_TYPE_ITSM              = "itsm"
	SERVICE_TYPE_VNCPROXY          = "vncproxy"
	SERVICE_TYPE_KEYSTONE          = "identity"
	SERVICE_TYPE_NOTIFY            = "notify"
	SERVICE_TYPE_CLOUDWATCHER      = "cloudwatcher"
	SERVICE_TYPE_MONITOR           = "monitor"
	SERVICE_TYPE_SERVICETREE       = "servicetree"
	SERVICE_TYPE_LOG               = "log"
	SERVICE_TYPE_REGION            = "compute"
	SERVICE_TYPE_SUGGESTION        = "suggestion"

	SERVICE_TYPE_ETCD     = "etcd"
	SERVICE_TYPE_INFLUXDB = "influxdb"
)

var (
	NO_RESOURCE_SERVICES = []string{
		SERVICE_TYPE_OFFLINE_CLOUDMETA,
		SERVICE_TYPE_CLOUDMETA,
		SERVICE_TYPE_WEBSOCKET,
		SERVICE_TYPE_AUTOUPDATE,
		SERVICE_TYPE_YUNIONAGENT,
		SERVICE_TYPE_SCHEDULER,
		SERVICE_TYPE_ITSM,
		SERVICE_TYPE_VNCPROXY,
		SERVICE_TYPE_KEYSTONE,
		SERVICE_TYPE_CLOUDWATCHER,
		SERVICE_TYPE_SERVICETREE,
		SERVICE_TYPE_ETCD,
		SERVICE_TYPE_INFLUXDB,
	}
)
