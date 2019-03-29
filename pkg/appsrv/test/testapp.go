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

package main

import (
	"context"
	"net/http"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
)

func main() {
	app := appsrv.NewApplication("test", 4)
	app.AddHandler("GET", "/delay", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 20)
		log.Debugf("end of delay sleep....")
		appsrv.Send(w, "pong")
	})
	app.AddHandler("GET", "/panic", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		panic("the handler is panic")
	})
	app.AddHandler("GET", "/delaypanic", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 1)
		panic("the handler is panic")
	})
	app.ListenAndServe("0.0.0.0:44444")
}
