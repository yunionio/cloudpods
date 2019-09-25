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

package utils

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func FetchOwnerId(ctx context.Context, manager db.IModelManager, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	var ownerId mcclient.IIdentityProvider
	var err error
	if manager.ResourceScope() != rbacutils.ScopeSystem {
		ownerId, err = manager.FetchOwnerId(ctx, data)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
	}
	if ownerId == nil {
		ownerId = userCred
	}
	return ownerId, nil
}

func GenerateMobileToken() string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	token := fmt.Sprintf("%06v", rnd.Int31n(1000000))
	return token
}

func GenerateEmailToken(tokenLen int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	token := new(bytes.Buffer)
	for token.Len() < tokenLen {
		token.WriteString(fmt.Sprintf("%x", rnd.Int31()))
	}
	return token.String()[:tokenLen]
}

func JsonArrayToStringArray(src []jsonutils.JSONObject) []string {
	des := make([]string, len(src))
	for i := range src {
		des[i], _ = src[i].GetString()
	}
	return des
}
