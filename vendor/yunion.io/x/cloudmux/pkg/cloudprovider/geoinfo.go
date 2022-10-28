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

// +onecloud:model-api-gen
type SGeographicInfo struct {
	// 纬度
	// example: 26.647003
	Latitude float32 `list:"user" update:"admin" create:"admin_optional"`
	// 经度
	// example: 106.630211
	Longitude float32 `list:"user" update:"admin" create:"admin_optional"`

	// 城市
	// example: Guiyang
	City string `list:"user" width:"32" update:"admin" create:"admin_optional"`
	// 国家代码
	// example: CN
	CountryCode string `list:"user" width:"4" update:"admin" create:"admin_optional"`
}
