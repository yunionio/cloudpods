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
	"net/http"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/cache"
	"yunion.io/x/pkg/util/s3auth"

	"yunion.io/x/onecloud/pkg/mcclient"
)

// maxSignatureSkew is the maximal clock skew accepted for a signed request,
// the same window AWS uses: requests signed earlier can not be replayed
// indefinitely.
const maxSignatureSkew = 15 * time.Minute

// checkRequestFreshness rejects replayed signatures: a signed request is only
// accepted within 15 minutes of its signing time.
func checkRequestFreshness(req http.Request) error {
	dateStr := req.Header.Get("x-amz-date")
	if len(dateStr) > 0 {
		signTime, err := time.Parse("20060102T150405Z", dateStr)
		if err != nil {
			return errors.Wrap(err, "invalid x-amz-date")
		}
		return checkTimeSkew(signTime)
	}
	dateStr = req.Header.Get("Date")
	if len(dateStr) > 0 {
		signTime, err := http.ParseTime(dateStr)
		if err != nil {
			return errors.Wrap(err, "invalid Date header")
		}
		return checkTimeSkew(signTime)
	}
	return errors.Error("missing signing date")
}

func checkTimeSkew(signTime time.Time) error {
	skew := time.Since(signTime)
	if skew > maxSignatureSkew || skew < -maxSignatureSkew {
		return errors.Errorf("request signature expired, signed at %s", signTime)
	}
	return nil
}

type sAccessKeyCache struct {
	*cache.LRUCache
}

type sAkSkCacheItem struct {
	credential *mcclient.SAkskTokenCredential
}

func (item *sAkSkCacheItem) Size() int {
	return 1
}

func newAccessKeyCache() *sAccessKeyCache {
	return &sAccessKeyCache{
		LRUCache: cache.NewLRUCache(defaultCacheCount),
	}
}

func (c *sAccessKeyCache) addToken(cred *mcclient.SAkskTokenCredential) {
	item := &sAkSkCacheItem{cred}
	c.Set(cred.AccessKeySecret.AccessKey, item)
}

func (c *sAccessKeyCache) getToken(token string) (*mcclient.SAkskTokenCredential, bool) {
	item, found := c.Get(token)
	if !found {
		return nil, false
	}
	return item.(*sAkSkCacheItem).credential, true
}

func (c *sAccessKeyCache) deleteToken(token string) bool {
	return c.Delete(token)
}

func (c *sAccessKeyCache) Verify(cli *mcclient.Client, req http.Request, virtualHost bool) (mcclient.TokenCredential, error) {
	aksk, err := s3auth.DecodeAccessKeyRequest(req, virtualHost)
	if err != nil {
		return nil, errors.Wrap(err, "s3auth.DecodeAccessKeyRequestV2")
	}
	if err := checkRequestFreshness(req); err != nil {
		return nil, errors.Wrap(err, "checkRequestFreshness")
	}

	token, found := c.getToken(aksk.GetAccessKey())
	if found {
		if token.Token.IsValid() && token.AccessKeySecret.IsValid() {
			err = aksk.Verify(token.AccessKeySecret.Secret)
			if err != nil {
				return nil, errors.Wrap(err, "aksk.Verify")
			}
			return token.Token, nil
		} else {
			c.deleteToken(aksk.GetAccessKey())
		}
	}

	token, err = cli.VerifyRequest(req, aksk, virtualHost)
	if err != nil {
		return nil, errors.Wrap(err, "cli.VerifyRequest")
	}

	c.addToken(token)

	return token.Token, nil
}
