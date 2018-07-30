package middleware

import (
	"fmt"
	"net/http"

	"gopkg.in/gin-gonic/gin.v1"

	"github.com/yunionio/mcclient/auth"
)

const (
	XAuthTokenKey = "X-Auth-Token"
)

func KeystoneTokenVerifyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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
