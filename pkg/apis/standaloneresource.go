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

type STag struct {
	Key   string
	Value string
}

type StandaloneResourceListInput struct {
	ResourceBaseListInput

	// filter by tags
	Tags []STag `json:"tags"`
	// Piggyback user metadata information
	WithoutUserMeta bool `json:"without_user_meta"`
	// Piggyback metadata information
	WithMeta *bool `json:"with_meta"`
	// Show all resources including the emulated resources
	ShowEmulated *bool `json:"show_emulated"`

	// filter by resource name
	Names []string `json:"name"`
	// filter by resource ID
	Ids []string `json:"id"`
}

type StatusStandaloneResourceListInput struct {
	StandaloneResourceListInput

	// filter by resource status
	Status []string `json:"status"`
}

type EnabledStatusStandaloneResourceListInput struct {
	StatusStandaloneResourceListInput

	// filter by enable/disabled status
	Enabled *bool `json:"enabled"`
}
