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

package handler

import (
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/ansible"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/cloudevent"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/cloudid"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/cloudnet"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/devtool"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/etcd"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/logger"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/quota"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/scheduledtask"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/webconsole"
	_ "yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
)

func init() {
	modules.InitUsages()
	modules.Usages.RegisterManager(modules.UsageManagerK8s, k8s.Usages)
}
