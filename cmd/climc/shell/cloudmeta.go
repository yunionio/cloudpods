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

package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	type CloudmetaOptions struct {
		PROVIDER_ID string `help:"provider_id"`
		REGION_ID   string `help:"region_id"`
		ZONE_ID     string `help:"zone_id"`
	}
	R(&CloudmetaOptions{}, "instance-type-list", "query backend service for its version", func(s *mcclient.ClientSession, args *CloudmetaOptions) error {
		return nil
	})
}
