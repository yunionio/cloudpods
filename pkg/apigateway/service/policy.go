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

package service

import (
	_ "yunion.io/x/onecloud/pkg/cloudevent/policy"
	compute "yunion.io/x/onecloud/pkg/compute/policy"
	image "yunion.io/x/onecloud/pkg/image/policy"
	keystone "yunion.io/x/onecloud/pkg/keystone/policy"
	logger "yunion.io/x/onecloud/pkg/logger/policy"
	notify "yunion.io/x/onecloud/pkg/notify/policy"
	yunionconf "yunion.io/x/onecloud/pkg/yunionconf/policy"
)

func InitDefaultPolicy() {
	compute.Init()
	image.Init()
	keystone.Init()
	logger.Init()
	notify.Init()
	yunionconf.Init()
}
