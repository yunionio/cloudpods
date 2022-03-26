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

package guesthandlers

import (
	"context"

	"yunion.io/x/jsonutils"

	hostapis "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func guestOpenForward(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	req := &hostapis.GuestOpenForwardRequest{}
	if err := body.Unmarshal(req); err != nil {
		return nil, httperrors.NewInputParameterError("unmarshal: %v", err)
	}
	gm := guestman.GetGuestManager()
	resp, err := gm.OpenForward(ctx, sid, req)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return jsonutils.Marshal(resp), nil
}

func guestListForward(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	req := &hostapis.GuestListForwardRequest{}
	if err := body.Unmarshal(req); err != nil {
		return nil, httperrors.NewInputParameterError("unmarshal: %v", err)
	}
	gm := guestman.GetGuestManager()
	resp, err := gm.ListForward(ctx, sid, req)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return jsonutils.Marshal(resp), nil
}

func guestCloseForward(ctx context.Context, userCred mcclient.TokenCredential, sid string, body jsonutils.JSONObject) (interface{}, error) {
	req := &hostapis.GuestCloseForwardRequest{}
	if err := body.Unmarshal(req); err != nil {
		return nil, httperrors.NewInputParameterError("unmarshal: %v", err)
	}
	gm := guestman.GetGuestManager()
	resp, err := gm.CloseForward(ctx, sid, req)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return jsonutils.Marshal(resp), nil
}
