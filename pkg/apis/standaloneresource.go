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

type StandaloneResourceShortDescDetail struct {
	ModelBaseShortDescDetail

	Id   string `json:"id"`
	Name string `json:"name"`
}

type StandaloneResourceListInput struct {
	ModelBaseListInput

	Tags            []string `json:"tags"`
	WithoutUserMeta bool     `json:"without_user_meta"`
}

type StandaloneResourceCreatInput struct {
	Meta
	// description: resource name
	// unique: true
	// required: true
	// example: yunion
	Name        string `json:"name"`
	Description string `json:"description"`
	IsEmulated  bool   `json:"is_emulated"`
}
