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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type CloudaccountCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	Provider            string
	AccountId           string
	Brand               string
	IsPublicCloud       bool
	IsOnPremise         bool
	Tenant              string
	TenantId            string
	EnableAutoSync      bool
	SyncIntervalSeconds int
	AutoCreateProject   bool
	Options             *jsonutils.JSONObject

	cloudprovider.SCloudaccount
	cloudprovider.SCloudaccountCredential
}

type CloudaccountShareModeInput struct {
	apis.Meta

	ShareMode string
}

func (i CloudaccountShareModeInput) Validate() error {
	if len(i.ShareMode) == 0 {
		return httperrors.NewMissingParameterError("share_mode")
	}
	if !utils.IsInStringArray(i.ShareMode, CLOUD_ACCOUNT_SHARE_MODES) {
		return httperrors.NewInputParameterError("invalid share_mode %s", i.ShareMode)
	}
	return nil
}
