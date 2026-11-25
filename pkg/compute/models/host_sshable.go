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
	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func (host *SHost) GetSshPort(ctx context.Context, userCred mcclient.TokenCredential) int {
	port := getSshPort(ctx, userCred, host)
	if port <= 0 {
		return 22
	}
	return port
}

func (host *SHost) PerformSetSshport(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input compute_api.GuestSetSshportInput) (jsonutils.JSONObject, error) {
	if input.Port <= 0 {
		return nil, httperrors.NewInputParameterError("invalid port")
	}
	return nil, setSshPort(ctx, userCred, host, input.Port)
}

func (host *SHost) GetDetailsSshport(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (compute_api.GuestSshportOutput, error) {
	port := host.GetSshPort(ctx, userCred)
	return compute_api.GuestSshportOutput{Port: port}, nil
}
