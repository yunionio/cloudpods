package handler

import (
	"net/http"

	"gopkg.in/gin-gonic/gin.v1"

	"github.com/yunionio/pkg/util/version"
)

func InstallVersionHandler(r *gin.Engine) {
	r.GET("/version", versionHandler)
}

func versionHandler(c *gin.Context) {
	c.String(http.StatusOK, version.GetShortString())
}
