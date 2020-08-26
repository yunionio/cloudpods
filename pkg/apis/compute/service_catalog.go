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

type ServiceCatalogCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	// description: service catalog icon url
	// example: https://yunion.io/files/hello.png
	IconUrl string `json:"icon_url"`

	// description: the id or name of guest template
	// example: good
	// required: true
	GuestTemplate string `json:"guest_template"`
}

type ServiceCatalogUpdateInput struct {
	// description: resource name
	// unique: true
	// example: test-network
	Name string `json:"name"`

	// description: service catalog icon url
	// example: https://yunion.io/files/hello.png
	IconUrl string `json:"icon_url"`

	// description: the id or name of guest template
	// example: good
	GuestTemplate string `json:"guest_template"`
}

type ServiceCatalogDeploy struct {

	// description: name of the new vm
	// example: hello
	Name string `json:"name"`

	// description: generate name automatically if name is repeated, and only one of name and this shoudle be given
	// example: hello
	GenerateName string `json:"generate_name"`

	// description: the count of the new vm
	// example: 1
	Count int `json:"count"`
}

type ServiceCatalogDetails struct {
	apis.SharableVirtualResourceDetails

	SServiceCatalog
}
