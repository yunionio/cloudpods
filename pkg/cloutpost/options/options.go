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

package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type CloudSyncOptions struct {
	Provider    string `help:"Public cloud provider" choices:"Aliyun|Azure|Aws|Qcloud|OpenStack|Ucloud|Huawei|ZStack"`
	Environment string `help:"environment of public cloud"`
	Cloudregion string `help:"region of public cloud"`
	Zone        string `help:"availability zone of public cloud"`

	etcd.SEtcdOptions

	common_options.CommonOptions
}

var (
	Options CloudSyncOptions
)
