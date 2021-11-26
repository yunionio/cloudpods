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

import (
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

// +onecloud:model-api-gen
type SCdnDomain struct {
	// cdn加速域名
	Domain string
	// 状态 rejected(域名未审核)|processing(部署中)|online|offline
	Status string
	// 区域 mainland|overseas|global
	Area string
	// cdn Cname
	Cname string
	// 源站
	Origin string
	// 源站类型 domain|ip|bucket
	OriginType string
}

// +onecloud:model-api-gen
type SCdnOrigin struct {
	// 源站类型
	// domain: 域名类型, cos：对象存储源站, ip：IP 列表作为源站
	// enmu: domain, cos, ip
	// required: true
	Type string
	// 源站地址
	Origin string
	// 回主源站时 Host 头部
	ServerName string
	// 回源协议
	// enmu: http, follow, https
	Protocol string
	Path     string
	Port     int
	Enabled  string
	Priority int
}

// +onecloud:model-api-gen
type SCdnOrigins []SCdnOrigin

func (self SCdnOrigins) IsZero() bool {
	return len(self) == 0
}

func (self SCdnOrigins) String() string {
	return jsonutils.Marshal(self).String()
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SCdnOrigins{}), func() gotypes.ISerializable {
		return &SCdnOrigins{}
	})
}

type CdnCreateOptions struct {
	Domain      string
	ServiceType string
	Area        string

	Origins SCdnOrigins
}
