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

package generic

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/redfish"
)

const (
	basePath   = "/redfish/v1"
	linkKey    = "@odata.id"
	memberKey  = "Members"
	versionKey = "RedfishVersion"
)

type SGenericRedfishApiFactory struct {
}

func (f *SGenericRedfishApiFactory) Name() string {
	return "Redfish"
}

func (f *SGenericRedfishApiFactory) NewApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	return NewGenericRedfishApi(endpoint, username, password, debug)
}

func init() {
	redfish.RegisterDefaultApiFactory(&SGenericRedfishApiFactory{})
}

type SGenericRefishApi struct {
	redfish.SBaseRedfishClient
}

func NewGenericRedfishApi(endpoint, username, password string, debug bool) redfish.IRedfishDriver {
	api := &SGenericRefishApi{
		SBaseRedfishClient: redfish.NewBaseRedfishClient(endpoint, username, password, debug),
	}
	api.SetVirtualObject(api)
	return api
}

func (r *SGenericRefishApi) BasePath() string {
	return basePath
}

func (r *SGenericRefishApi) GetParent(parent jsonutils.JSONObject) jsonutils.JSONObject {
	return parent
}

func (r *SGenericRefishApi) VersionKey() string {
	return versionKey
}

func (r *SGenericRefishApi) LinkKey() string {
	return linkKey
}

func (r *SGenericRefishApi) MemberKey() string {
	return memberKey
}

func (r *SGenericRefishApi) LogItemsKey() string {
	return memberKey
}
