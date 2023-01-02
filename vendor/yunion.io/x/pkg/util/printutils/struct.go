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

package printutils

import (
	"yunion.io/x/jsonutils"
)

type ListResult struct {
	Data []jsonutils.JSONObject `json:"data,allowempty"`

	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	Totals jsonutils.JSONObject `json:"totals"`

	NextMarker  string
	MarkerField string
	MarkerOrder string
}

type SubmitResult struct {
	Status int
	Id     interface{}
	Data   jsonutils.JSONObject
}
