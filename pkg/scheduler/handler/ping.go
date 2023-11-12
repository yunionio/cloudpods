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

package handler

import (
	"fmt"
	"net/http"

	gin "github.com/gin-gonic/gin"

	"yunion.io/x/log"

	schedman "yunion.io/x/onecloud/pkg/scheduler/manager"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
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

	c.String(http.StatusOK, "pong")
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
		o.Options.DisableBaremetalPredicates = true
		result = "ignore_baremetal_filter_switch is true"
	} else {
		o.Options.DisableBaremetalPredicates = false
		result = "ignore_baremetal_filter_switch is false"
	}

	c.JSON(http.StatusOK, result)
}
