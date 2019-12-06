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

package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	Cloudevents modulebase.ResourceManager
)

func init() {
	Cloudevents = NewCloudeventManager("cloudevent", "cloudevents",
		[]string{"ID", "Name", "Status", "Service", "Success",
			"Resource_Type", "Action", "Cloudprovider_Id", "Cloudprovider"},
		[]string{})

	register(&Cloudevents)
}
