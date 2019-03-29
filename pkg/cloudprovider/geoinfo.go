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

package cloudprovider

type SGeographicInfo struct {
	Latitude  float32 `list:"user" update:"admin" create:"admin_optional"`
	Longitude float32 `list:"user" update:"admin" create:"admin_optional"`

	City        string `list:"user" width:"32" update:"admin" create:"admin_optional"`
	CountryCode string `list:"user" width:"4" update:"admin" create:"admin_optional"`
}
