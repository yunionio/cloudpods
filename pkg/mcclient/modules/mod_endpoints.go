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
	Endpoints   modulebase.ResourceManager
	EndpointsV3 modulebase.ResourceManager
)

func init() {
	Endpoints = NewIdentityManager("endpoint", "endpoints",
		[]string{},
		[]string{"ID", "Region", "Zone",
			"Service_ID", "Service_name",
			"PublicURL", "AdminURL", "InternalURL"})

	register(&Endpoints)

	EndpointsV3 = NewIdentityV3Manager("endpoint", "endpoints",
		[]string{},
		[]string{"ID", "Region_ID",
			"Service", "Service_ID", "Service_Name", "Service_Type",
			"URL", "Interface", "Enabled"})

	register(&EndpointsV3)
}
