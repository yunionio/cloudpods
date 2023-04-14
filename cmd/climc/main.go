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

package main

import (
	"yunion.io/x/onecloud/cmd/climc/entry"
	_ "yunion.io/x/onecloud/cmd/climc/shell"
	_ "yunion.io/x/onecloud/cmd/climc/shell/ansible"
	_ "yunion.io/x/onecloud/cmd/climc/shell/apimap"
	_ "yunion.io/x/onecloud/cmd/climc/shell/cloudevent"
	_ "yunion.io/x/onecloud/cmd/climc/shell/cloudid"
	_ "yunion.io/x/onecloud/cmd/climc/shell/cloudnet"
	_ "yunion.io/x/onecloud/cmd/climc/shell/cloudproxy"
	_ "yunion.io/x/onecloud/cmd/climc/shell/compute"
	_ "yunion.io/x/onecloud/cmd/climc/shell/devtool"
	_ "yunion.io/x/onecloud/cmd/climc/shell/etcd"
	_ "yunion.io/x/onecloud/cmd/climc/shell/events"
	_ "yunion.io/x/onecloud/cmd/climc/shell/identity"
	_ "yunion.io/x/onecloud/cmd/climc/shell/image"
	_ "yunion.io/x/onecloud/cmd/climc/shell/k8s"
	_ "yunion.io/x/onecloud/cmd/climc/shell/logger"
	_ "yunion.io/x/onecloud/cmd/climc/shell/misc"
	_ "yunion.io/x/onecloud/cmd/climc/shell/monitor"
	_ "yunion.io/x/onecloud/cmd/climc/shell/notifyv2"
	_ "yunion.io/x/onecloud/cmd/climc/shell/quota"
	_ "yunion.io/x/onecloud/cmd/climc/shell/scheduledtask"
	_ "yunion.io/x/onecloud/cmd/climc/shell/scheduler"
	_ "yunion.io/x/onecloud/cmd/climc/shell/webconsole"
	_ "yunion.io/x/onecloud/cmd/climc/shell/yunionconf"
)

func main() {
	entry.ClimcMain()
}
