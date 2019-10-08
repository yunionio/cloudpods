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
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SZoneResourceBase struct {
	ZoneId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

func (self *SZoneResourceBase) GetZone() *SZone {
	return ZoneManager.FetchZoneById(self.ZoneId)
}

func (self *SZoneResourceBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	zone := self.GetZone()
	if zone == nil {
		return nil
	}
	info := map[string]string{
		"zone":    zone.GetName(),
		"zone_id": zone.GetId(),
	}
	if len(zone.ExternalId) > 0 {
		info["zone_ext_id"] = fetchExternalId(zone.ExternalId)
	}
	if region := zone.GetRegion(); region != nil {
		info["region"] = region.GetName()
		info["region_id"] = region.GetId()
		if len(region.ExternalId) > 0 {
			info["region_ext_id"] = fetchExternalId(region.ExternalId)
		}
	}
	return jsonutils.Marshal(info).(*jsonutils.JSONDict)
}
