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
	"net/http"

	gin "gopkg.in/gin-gonic/gin.v1"

	"yunion.io/x/pkg/util/version"
)

func InstallVersionHandler(r *gin.Engine) {
	r.GET("/version", versionHandler)
}

func versionHandler(c *gin.Context) {
	c.String(http.StatusOK, version.GetShortString())
}
