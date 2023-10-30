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

package models

import (
	"strconv"
	"strings"

	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/pinyinutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SHostnameResourceBase struct {
	Hostname string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
}

type SHostnameResourceBaseManager struct {
}

func (manager *SHostnameResourceBaseManager) ValidateHostname(name string, osType string, input api.HostnameInput) (api.HostnameInput, error) {
	inputHostname := true
	if len(input.Hostname) == 0 {
		inputHostname = false
		if len(name) == 0 {
			return input, httperrors.NewMissingParameterError("name")
		}
		input.Hostname = pinyinutils.Text2Pinyin(name)
	}
	hostname := ""
	for _, s := range input.Hostname {
		if (s >= '0' && s <= '9') || (s >= 'a' && s <= 'z') || (s >= 'A' && s <= 'Z') || strings.Contains(".-", string(s)) {
			hostname += string(s)
		}
	}
	input.Hostname = hostname
	for strings.HasPrefix(input.Hostname, ".") || strings.HasPrefix(input.Hostname, "-") ||
		strings.HasSuffix(input.Hostname, ".") || strings.HasSuffix(input.Hostname, "-") ||
		strings.Contains(input.Hostname, "..") || strings.Contains(input.Hostname, "--") {
		input.Hostname = strings.TrimPrefix(input.Hostname, ".")
		input.Hostname = strings.TrimPrefix(input.Hostname, "-")
		input.Hostname = strings.TrimSuffix(input.Hostname, ".")
		input.Hostname = strings.TrimSuffix(input.Hostname, "-")
		input.Hostname = strings.ReplaceAll(input.Hostname, "--", "")
		input.Hostname = strings.ReplaceAll(input.Hostname, "..", "")
	}
	if len(input.Hostname) > 60 {
		input.Hostname = input.Hostname[:60]
	}
	if strings.EqualFold(osType, osprofile.OS_TYPE_WINDOWS) {
		if num, err := strconv.Atoi(input.Hostname); err == nil && num > 0 {
			return input, httperrors.NewInputParameterError("hostname cannot be number %d", num)
		}
		input.Hostname = strings.ReplaceAll(input.Hostname, ".", "")
		if len(input.Hostname) > api.MAX_WINDOWS_COMPUTER_NAME_LENGTH {
			if inputHostname {
				return input, httperrors.NewInputParameterError("Windows hostname cannot be longer than %d characters", api.MAX_WINDOWS_COMPUTER_NAME_LENGTH)
			}
			input.Hostname = input.Hostname[:15]
		}
	}
	for strings.HasSuffix(input.Hostname, "-") {
		input.Hostname = strings.TrimSuffix(input.Hostname, "-")
	}
	if len(input.Hostname) < 2 {
		return input, httperrors.NewInputParameterError("the hostname length must be greater than or equal to 2")
	}
	return input, nil
}
