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

package auth

import (
	"context"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type TAuthHook func(ctx context.Context, userCred mcclient.TokenCredential)

var (
	authHooks = make([]TAuthHook, 0)
)

func RegisterAuthHook(hook TAuthHook) {
	authHooks = append(authHooks, hook)
}

func callbackAuthhooks(ctx context.Context, userCred mcclient.TokenCredential) {
	for i := range authHooks {
		authHooks[i](ctx, userCred)
	}
}
