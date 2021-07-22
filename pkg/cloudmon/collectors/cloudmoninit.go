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

package collectors

import (
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/alertrecordhistorymon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/alimon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/apsaramon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/awsmon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/azuremon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/cloudaccountmon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/gcpmon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/huaweimon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/jdmon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/qcmon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/storagemon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/vmwaremon"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors/zstackmon"
	_ "yunion.io/x/onecloud/pkg/multicloud/aliyun/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/apsara/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/aws/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/azure/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/esxi/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/google/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/huawei/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/jdcloud/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/qcloud/provider"
	_ "yunion.io/x/onecloud/pkg/multicloud/zstack/provider"
)
