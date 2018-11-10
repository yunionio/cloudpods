package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/gin-gonic/gin.v1"

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

		_, err := auth.Verify(token)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}
		c.Next()
	}
}
