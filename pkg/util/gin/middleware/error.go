package middleware

import (
	"gopkg.in/gin-gonic/gin.v1"
)

func ErrorHandler(c *gin.Context) {
	c.Next()

	if c.Errors.Last() != nil {
		c.JSON(-1, gin.H{"code": c.Writer.Status(), "details": c.Errors.String()}) // -1 == not override the current error coe
	}
}
