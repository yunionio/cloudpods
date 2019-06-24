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

	"github.com/gin-gonic/gin"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
)

func SendJSON(c *gin.Context, code int, obj interface{}) {
	c.Render(http.StatusOK, JSON{Data: obj})
}

type JSON struct {
	Data interface{}
}

func (r JSON) Render(w http.ResponseWriter) error {
	appsrv.SendJSON(w, jsonutils.Marshal(r.Data))
	return nil
}

func (r JSON) WriteContentType(w http.ResponseWriter) {
}
