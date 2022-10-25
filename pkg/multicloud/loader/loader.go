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

package loader

import (
	"yunion.io/x/log" // on-premise virtualization technologies

	_ "yunion.io/x/onecloud/pkg/multicloud/aliyun/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/apsara/provider" // aliyun apsara stack
	_ "yunion.io/x/onecloud/pkg/multicloud/aws/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/azure/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/bingocloud/provider" // private clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/cloudpods/provider"  // private clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/ctyun/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/ecloud/provider" // public clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/esxi/provider"   // private clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/google/provider" // public clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/hcs/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/hcso/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/huawei/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/incloudsphere/provider" // private clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/jdcloud/provider"       // public clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/nutanix/provider"       // private clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/objectstore/ceph/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/objectstore/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/objectstore/xsky/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/openstack/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/proxmox/provider" // private clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/qcloud/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/remotefile/provider" // private clouds
	_ "yunion.io/x/onecloud/pkg/multicloud/ucloud/provider"     // object storages
	_ "yunion.io/x/onecloud/pkg/multicloud/zstack/provider"     // private clouds
)

func init() {
	log.Infof("Loading cloud providers ...")
}
