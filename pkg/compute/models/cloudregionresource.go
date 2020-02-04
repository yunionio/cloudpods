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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCloudregionResourceBase struct {
	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"user" default:"default" create:"optional"`
}

func (self *SCloudregionResourceBase) GetRegion() *SCloudregion {
	region, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		log.Errorf("failed to find cloudregion %s error: %v", self.CloudregionId, err)
		return nil
	}
	return region.(*SCloudregion)
}

func (self *SCloudregionResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) api.CloudregionInfo {
	out := api.CloudregionInfo{}
	region := self.GetRegion()
	if region == nil {
		return out
	}
	return region.GetRegionInfo()
}
