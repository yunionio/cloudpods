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

package cache

import (
	"time"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/util/hashcache"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	tokenCache *hashcache.Cache
)

func Init(expire int) {
	tokenCache = hashcache.NewCache(2048, time.Duration(expire/2)*time.Second)
}

func Save(tokenStr string, token interface{}) {
	tokenCache.AtomicSet(tokenStr, token)
}

func Remove(tokenStr string) {
	tokenCache.AtomicRemove(tokenStr)
}

func Get(tokenStr string) interface{} {
	if len(tokenStr) > api.AUTH_TOKEN_LENGTH {
		// hash
		tokenStr = stringutils2.GenId(tokenStr)
	}
	return tokenCache.AtomicGet(tokenStr)
}
