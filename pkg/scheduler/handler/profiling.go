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
	"net/http/pprof"

	gin "github.com/gin-gonic/gin"
)

// InstallProfiling 在 gin 引擎上注册 pprof 相关路由。
// 路由与 appsrv 中保持一致：/debug/pprof, /cmdline, /profile, /symbol, /trace
func InstallProfiling(r *gin.Engine, enableProfiling bool) {
	if !enableProfiling {
		return
	}

	base := "/debug/pprof"

	// 与 net/http/pprof 的默认注册保持一致
	r.GET(base, gin.WrapF(pprof.Index))
	r.GET(base+"/", gin.WrapF(pprof.Index))
	r.GET(base+"/cmdline", gin.WrapF(pprof.Cmdline))
	r.GET(base+"/profile", gin.WrapF(pprof.Profile))
	r.GET(base+"/symbol", gin.WrapF(pprof.Symbol))
	r.POST(base+"/symbol", gin.WrapF(pprof.Symbol))
	r.GET(base+"/trace", gin.WrapF(pprof.Trace))
}
