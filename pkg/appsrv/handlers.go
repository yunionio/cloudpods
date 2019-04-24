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

package appsrv

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/pkg/util/version"
)

type FilterHandler func(ctx context.Context, w http.ResponseWriter, r *http.Request)

func VersionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, version.GetShortString())
}

func PingHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

/*
func CORSHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    reqHdrs, enableCors := r.Header["Access-Control-Request-Headers"]
    if enableCors {
        w.Header().Set("Access-Control-Allow-Origin", getRequestOrigin(r))
        allowHdrs := strings.Join(reqHdrs, ",")
        allowHdrs = fmt.Sprintf("%s,%s", allowHdrs, "Authorization")
        w.Header().Set("Vary", "Origin,Access-Control-Request-Method,Access-Control-Request-Headers")
        w.Header().Set("Access-Control-Allow-Headers", allowHdrs)
        w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,PUT,PATCH,DELETE")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Expose-Headers", allowHdrs)
        w.Header().Set("Access-Control-Max-Age", "86400")
    }
}*/
