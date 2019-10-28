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

import (
	"yunion.io/x/onecloud/pkg/apis/compute"
)

// swagger:route POST /servers instance serverCreateInput
//
// Create instance server
//
// Create instance server by hypervisor
//
// responses:
// 200: serverOutput

// swagger:parameters serverCreateInput
type serverCreateInput struct {
	// in:body
	// required: true
	Server compute.ServerCreateInput `json:"server"`
}

// swagger:response serverOutput
type serverResponse struct {
	// in:body
	Body struct {
		Output compute.ServerCreateOutput `json:"server"`
	}
}

// swagger:route GET /servers instance serverListInput
//
// List instance servers
//
// responses:
// 200: serverListOutput

// swagger:parameters serverListInput
type serverListInput struct {
	compute.ServerListInput
}

// swagger:response serverListOutput
type serverListOutput struct {
	// in:body
	Body struct {
		Output compute.ServerCreateOutput `json:"servers"`
	}
}

// swagger:route PUT /servers/{id} instance serverUpdateInput
//
// Update server
//
// responses:
// 200: serverOutput

// swagger:parameters serverUpdateInput
type serverUpdateInput struct {
	objectId

	// in:body
	Body compute.ServerUpdateInput `json:"server"`
}

// swagger:route GET /servers/{id} instance serverShowInput
//
// Get server details
//
// responses:
// 200: serverOutput

// swagger:parameters serverShowInput
type serverShowInput struct {
	objectId
}

// swagger:route DELETE /servers/{id} instance serverDeleteInput
//
// Delete specific server
//
// responses:
// 200: serverOutput

// swagger:parameters serverDeleteInput
type serverDeleteInput struct {
	objectId
	compute.ServerDeleteInput
}

// swagger:route POST /servers/{id}/stop instance serverStop
//
// Stop server
//
// responses:
// 200: serverOutput

// swagger:parameters serverStop
type serverStopInput struct {
	objectId
	// in:body
	Input compute.ServerStopInput `json:"server"`
}
