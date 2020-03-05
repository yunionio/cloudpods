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

package feishu

import (
	"os"
	"testing"
	"time"
)

func TestFileCache(t *testing.T) {
	token := "atokeexamplenomeaning"
	cache := NewFileCache(".test_auth_file")
	defer func() {
		os.Remove(".test_auth_file")
	}()
	t.Run("unexpired", func(t *testing.T) {
		tokenIn := TenantAccesstoken{
			TenantAccessToken: token,
			Expire:            3600,
			Created:           time.Now().Unix(),
		}
		err := cache.Set(tokenIn)
		if err != nil {
			t.Fatalf("cache set error: %s", err)
		}
		var tokenOut TenantAccesstoken
		err = cache.Get(&tokenOut)
		if err != nil {
			t.Fatalf("cache get error: %s", err)
		}
		if tokenIn.TenantAccessToken != tokenOut.TenantAccessToken {
			t.Fatalf("the token value stored in cache is incorrect")
		}
	})

	t.Run("expired", func(t *testing.T) {
		tokenIn := TenantAccesstoken{
			TenantAccessToken: token,
			Expire:            119,
			Created:           time.Now().Unix(),
		}
		err := cache.Set(tokenIn)
		if err != nil {
			t.Fatalf("cache set error: %s", err)
		}
		var tokenOut TenantAccesstoken
		err = cache.Get(&tokenOut)
		if err == nil {
			t.Fatalf("Getting expired token from cache should produce an error")
		}
	})
}
