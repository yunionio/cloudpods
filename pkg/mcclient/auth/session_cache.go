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
	"sync"
	"time"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SessionCache struct {
	mu      sync.RWMutex
	session *mcclient.ClientSession

	Region     string
	APIVersion string

	// EarlyRefresh tells the cache how early to fetch a new session before
	// actual expiration of the old
	EarlyRefresh time.Duration

	Token         mcclient.TokenCredential
	UseAdminToken bool
}

func (sc *SessionCache) getToken() mcclient.TokenCredential {
	if sc.Token != nil {
		return sc.Token
	}
	if sc.UseAdminToken {
		return manager.adminCredential
	}
	return nil
}

func (sc *SessionCache) Get(ctx context.Context) *mcclient.ClientSession {
	mu := &sc.mu

	{
		mu.RLock()
		s := sc.session
		mu.RUnlock()

		if s != nil {
			token := s.GetToken()
			expires := token.GetExpires()
			if time.Now().Add(sc.EarlyRefresh).After(expires) {
				return s
			}
		}
	}

	mu.Lock()
	defer mu.Unlock()
	token := sc.getToken()
	sc.session = GetSession(ctx, token, sc.Region, sc.APIVersion)
	return sc.session
}
