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

package qcloud

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

func (client *SQcloudClient) AddCdnDomain(domain string, originType string, origins []string, cosPrivateAccess string) error {
	params := map[string]string{}
	params["Domain"] = domain
	params["ServiceType"] = "web"
	for i := range origins {
		params[fmt.Sprintf("Origin.Origins.%d", i)] = origins[i]
	}
	params["Origin.OriginType"] = originType
	params["Origin.CosPrivateAccess"] = cosPrivateAccess
	_, err := client.cdnRequest("AddCdnDomain", params)
	if err != nil {
		return errors.Wrapf(err, ` client.cdnRequest("AddCdnDomain", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}
