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

package middleware

import (
	"fmt"
	"net/http"
	"strings"

	gin "github.com/gin-gonic/gin"

	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	XAuthTokenKey = "X-Auth-Token"
)

func KeystoneTokenVerifyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// hack
		escapeAuth := []string{"ping", "version", "metrics", "k8s/predicates", "k8s/priorities"}
		for _, s := range escapeAuth {
			if strings.HasSuffix(c.Request.URL.Path, s) {
				c.Next()
				return
			}
		}

		token := c.Request.Header.Get(XAuthTokenKey)
		if len(token) == 0 {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Not found %s in http header.", XAuthTokenKey))
			return
		}

		_, err := auth.Verify(c, token)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}
		c.Next()
	}
}
