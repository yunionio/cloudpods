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
	"strings"

	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type TIpSetType string

const (
	IpSetTypeIpv4CidrList TIpSetType = "ipv4_cidr_list"
	IpSetTypeIpv6CidrList TIpSetType = "ipv6_cidr_list"
)

func (t TIpSetType) IsValid() bool {
	switch t {
	case IpSetTypeIpv4CidrList, IpSetTypeIpv6CidrList:
		return true
	default:
		return false
	}
}

type IpSetCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	// IP集合类型
	// enum: ["ipv4_cidr_list", "ipv6_cidr_list"]
	// required: true
	IpSetType TIpSetType `json:"ip_set_type"`

	// IP/CIDR 列表，逗号分隔
	// example: 192.168.1.0/24,10.0.0.1
	// required: true
	Data string `json:"data"`
}

type IpSetUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput

	// IP/CIDR 列表，逗号分隔
	// example: 192.168.1.0/24,10.0.0.1
	Data *string `json:"data"`
}

type IpSetListInput struct {
	apis.SharableVirtualResourceListInput

	// 按 IP 集合类型过滤
	// enum: ["ipv4_cidr_list", "ipv6_cidr_list"]
	IpSetType []TIpSetType `json:"ip_set_type"`

	// 根据 IP/CIDR 模糊匹配
	Ip string `json:"ip"`
}

type IpSetDetails struct {
	apis.SharableVirtualResourceDetails

	// 关联安全组数量
	SecurityGroupCount int `json:"security_group_count"`
}

func ValidateIpSetData(ipSetType TIpSetType, data string) error {
	if !ipSetType.IsValid() {
		return httperrors.NewInputParameterError("invalid ip_set_type %s", ipSetType)
	}
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		return httperrors.NewMissingParameterError("data")
	}
	parts := strings.Split(data, ",")
	for i := range parts {
		cidr := strings.TrimSpace(parts[i])
		if len(cidr) == 0 {
			return httperrors.NewInputParameterError("empty cidr in data")
		}
		switch ipSetType {
		case IpSetTypeIpv4CidrList:
			if !regutils.MatchCIDR(cidr) && !regutils.MatchIP4Addr(cidr) {
				return httperrors.NewInputParameterError("invalid ipv4 cidr or address %s", cidr)
			}
		case IpSetTypeIpv6CidrList:
			if !regutils.MatchCIDR6(cidr) && !regutils.MatchIP6Addr(cidr) {
				return httperrors.NewInputParameterError("invalid ipv6 cidr or address %s", cidr)
			}
		}
	}
	return nil
}

func (input *IpSetCreateInput) Validate() error {
	return ValidateIpSetData(input.IpSetType, input.Data)
}

func (input *IpSetUpdateInput) Validate(currentType TIpSetType) error {
	if input.Data != nil {
		return ValidateIpSetData(currentType, *input.Data)
	}
	return nil
}
