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

package tasks

import (
	_ "yunion.io/x/onecloud/pkg/compute/tasks/access_group"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/ai_gateway"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/app"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/backup"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/baremetal"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/bucket"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/cdn"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/cloudaccount"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/container"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/dbinstance"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/disk"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/dnszone"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/eip"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/elastic_search"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/elasticcache"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/filesystem"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/global_vpc"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/guest"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/host"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/inter_vpc"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/kafka"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/kube"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/loadbalancer"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/modelarts"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/mongodb"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/nat"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/network"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/route_table"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/scaling_group"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/security_group"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/server_sku"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/snapshot"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/snapshotpolicy"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/storage"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/vpc"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/vpc_peering"
	_ "yunion.io/x/onecloud/pkg/compute/tasks/waf"
)
