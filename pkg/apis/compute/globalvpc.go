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
	"yunion.io/x/onecloud/pkg/apis"
)

type GlobalVpcCreateInput struct {
	apis.Meta

	// description: global vpc name
	// unique: true
	// required: true
	// example: test-globalvpc
	Name string `json:"name"`

	// description: global vpc description
	// required: false
	// example: test create globalvpc
	Description string `json:"description"`

	// description: enable or disable global vpc
	// required: false
	// default: false
	Enabled *bool `json:"enabled"`

	// description: global vpc status
	// required: false
	// default: available
	Status string `json:"status"`
}
