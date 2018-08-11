package handler

import (
	"fmt"
	"net/http"

	"gopkg.in/gin-gonic/gin.v1"

	"yunion.io/x/log"
	o "yunion.io/x/onecloud/cmd/scheduler/options"
	schedman "yunion.io/x/onecloud/pkg/scheduler/manager"
)

var counter = 0

func InstallPingHandler(r *gin.Engine) {
	r.GET("/ping", pingHandler)
	r.GET("/switch", switchHandler)
}

// pingHandler is a handler of ping-pong service that just for
// testing scheduler serevice.
func pingHandler(c *gin.Context) {
	if !schedman.IsReady() {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Global scheduler not init"))
		return
	}
	log.Infof("%v", c.Request.Body)

	c.JSON(http.StatusOK, "pong")
}

// switchHandler is a handler of switch service that just for
// testing scheduler serevice.
func switchHandler(c *gin.Context) {
	if !schedman.IsReady() {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Global scheduler not init"))
		return
	}
	log.Infof("%v", c.Request.Body)
	counter++
	var result string
	if counter%2 == 1 {
		o.GetOptions().DisableBaremetalPredicates = true
		result = fmt.Sprintf("ignore_baremetal_filter_switch is true")
	} else {
		o.GetOptions().DisableBaremetalPredicates = false
		result = fmt.Sprintf("ignore_baremetal_filter_switch is false")
	}

	c.JSON(http.StatusOK, result)
}
