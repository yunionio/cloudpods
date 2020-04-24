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
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
type SVCenterManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var VCenterManager *SVCenterManager

func init() {
	VCenterManager = &SVCenterManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SVCenter{},
			"vcenters_tbl",
			"vcenter",
			"vcenters",
		),
	}
	VCenterManager.SetVirtualObject(VCenterManager)
}

type SVCenter struct {
	db.SEnabledStatusStandaloneResourceBase

	Hostname string `width:"64" charset:"ascii" nullable:"false" list:"admin"` // = Column(VARCHAR(64, charset='ascii'), nullable=False)
	Port     int    `nullable:"false" list:"admin"`                            // = Column(Integer, nullable=False)
	Account  string `width:"64" charset:"ascii" nullable:"false" list:"admin"` // = Column(VARCHAR(64, charset='ascii'), nullable=False)
	Password string `width:"256" charset:"ascii" nullable:"false"`             // = Column(VARCHAR(256, charset='ascii'), nullable=False)

	LastSync time.Time `nullable:"true" get:"admin"` // = Column(DateTime, nullable=True)

	Version string `width:"32" charset:"ascii" nullable:"true" list:"admin"` // = Column(VARCHAR(32, charset='ascii'), nullable=True)

	Sysinfo jsonutils.JSONObject `nullable:"true" get:"admin"` // = Column(JSONEncodedDict, nullable=True)
}

func (manager *SVCenterManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}
