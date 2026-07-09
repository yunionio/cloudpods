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

package yunionconf

import "yunion.io/x/onecloud/pkg/apis"

type TagListInput struct {
	apis.InfrasResourceBaseListInput
}

type TagDetails struct {
	apis.InfrasResourceBaseDetails
}

type TagCreateInput struct {
	apis.InfrasResourceBaseCreateInput

	Values []string `json:"values"`
}

type TagBatchImportInput struct {
	// 待导入的标签列表
	Tags []TagCreateInput `json:"tags"`
}

type TagBatchImportResult struct {
	// 新建的标签数量
	Created int `json:"created"`
	// 合并 values 的已有标签数量
	Updated int `json:"updated"`
	// 无需变更的标签数量
	Unchanged int `json:"unchanged"`
}
