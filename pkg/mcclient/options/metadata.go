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

type MetadataListOptions struct {
	Resources []string `help:"list of resource e.g server、disk、eip、snapshot, empty will show all metadata"`
	Service   string   `help:"service type"`

	SysMeta   *bool `help:"Show sys metadata only"`
	CloudMeta *bool `help:"Show cloud metadata olny"`
	UserMeta  *bool `help:"Show user metadata olny"`

	Admin *bool `help:"Show all metadata"`

	WithSysMeta   *bool `help:"Show sys metadata"`
	WithCloudMeta *bool `help:"Show cloud metadata"`
	WithUserMeta  *bool `help:"Show user metadata"`

	Key   []string `help:"key"`
	Value []string `help:"value"`

	Limit  *int `help:"limit"`
	Offset *int `help:"offset"`
}

type TagListOptions MetadataListOptions
