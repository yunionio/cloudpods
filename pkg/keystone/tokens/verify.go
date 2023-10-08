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

package tokens

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func FernetTokenVerifier(ctx context.Context, tokenStr string) (mcclient.TokenCredential, error) {
	token, err := TokenStrDecode(ctx, tokenStr)
	if err != nil {
		return nil, httperrors.NewInvalidCredentialError("invalid token %s", err)
	}
	userCred, err := token.GetSimpleUserCred(tokenStr)
	if err != nil {
		return nil, err
	}
	log.Debugf("FernetTokenVerify %s %#v %#v", tokenStr, token, userCred)
	return userCred, nil
}
