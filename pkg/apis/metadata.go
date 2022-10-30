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

import "yunion.io/x/cloudmux/pkg/apis"

const (
	IS_SYSTEM = apis.IS_SYSTEM
)

type MetadataListInput struct {
	ModelBaseListInput

	ScopedResourceInput

	DomainizedResourceInput

	ProjectizedResourceInput

	MetadataBaseFilterInput

	// 按关联资源类型过滤
	Resources []string `json:"resources"`
}

type MetadataBaseFilterInput struct {
	// 仅显示系统标签
	SysMeta *bool `json:"sys_meta"`

	// 仅显示同步下来的标签
	CloudMeta *bool `json:"cloud_meta"`

	// 仅显示用户标签
	UserMeta *bool `json:"user_meta"`

	// 同时显示系统标签
	WithSysMeta *bool `json:"with_sys_meta"`

	// 同时显示同步下来的标签
	WithCloudMeta *bool `json:"with_cloud_meta"`

	// 同时显示用户标签
	WithUserMeta *bool `json:"with_user_meta"`

	// 按Key过滤
	Key []string `json:"key"`

	// 按Value过滤
	Value []string `json:"value"`
}

type MetaGetPropertyTagValuePairsInput struct {
	MetadataListInput

	// 只输入Key
	KeyOnly *bool `json:"key_only"`
}
