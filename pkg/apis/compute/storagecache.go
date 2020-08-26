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

package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type StoragecacheDetails struct {
	apis.StandaloneResourceDetails
	ManagedResourceInfo

	SStoragecache

	// 存储列表
	Storages []string `json:"storages"`
	// 缓存镜像总大小
	Size int64 `json:"size"`
	// 缓存镜像个数
	Count int `json:"count"`
	// 通过一致性哈希获取的一个管理宿主机信息
	Host *jsonutils.JSONDict `json:"host"`
}

type StoragecacheListInput struct {
	apis.StandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput

	// 路径过滤
	Path []string `json:"path"`
}
