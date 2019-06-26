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

/*
Yunion Conf Service
“参数服务”在服务器端为指定用户持久化存储和管理个性化参数，例如 控制台的配置，列表的colume配置等，从而实现产品的个性化配置。
*/

import (
	"yunion.io/x/onecloud/pkg/util/atexit"
	"yunion.io/x/onecloud/pkg/yunionconf/service"
)

func main() {
	defer atexit.Handle()

	service.StartService()
}
