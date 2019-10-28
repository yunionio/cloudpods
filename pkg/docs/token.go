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

package docs

import "yunion.io/x/onecloud/pkg/apis/identity"

// swagger:route POST /v2.0/tokens authentication authV2Input
//
// keystone v2 authenticate api, auth by username/password or token string
//
// keystone v2 authentication
//
// responses:
// 200: authV2Output

// swagger:parameters authV2Input
type authV2Input struct {
	// in:body
	// required: true
	Auth identity.AuthV2Input
}

// swagger:response authV2Output
type authV2Output struct {
	// in:body
	Body struct {
		Access identity.TokenV2 `json:"access"`
	}
}
