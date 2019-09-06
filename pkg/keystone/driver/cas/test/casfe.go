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
	"fmt"
	"net/http"
	"os"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

var (
	authUrl    = "" // http://10.168.222.252:35357/v3
	casServer  = "" // https://cas.example.org/cas
	serviceUrl = "" // https://app.example.com
)

func defaultPage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		httperrors.BadRequestError(w, "parse query fail %s", err)
		return
	}
	if query.Contains("ticket") {
		ticket, _ := query.GetString("ticket")
		service := r.URL.Path
		referer := r.Header.Get("Referer")
		cliIp := netutils2.GetHttpRequestIp(r)

		cli := mcclient.NewClient(authUrl, 120, true, true, "", "")
		token, err := cli.AuthenticateCAS(ticket, "", "", "", cliIp)
		if err != nil {
			httperrors.InvalidCredentialError(w, "cas auth error %s", err)
			return
		}
		appsrv.SendHTML(w, fmt.Sprintf("<html><h1>Welcome</h1><h2>[%s]</h2><h2>[%s]</h2><h2>[%s]</h2><h2>[%s]</h2></html>", ticket, service, referer, token.GetUserName()))
		return
	} else {
		httperrors.HTTPError(w, fmt.Sprintf("%s/login?service=%s", casServer, serviceUrl), 302, "Redirect", httputils.Error{})
		return
	}
}

func main() {
	if len(os.Args) <= 6 {
		fmt.Printf("usage: %s <authUrl> <casServer> <serviceUrl> <certfile> <keyfile> <port>\n", os.Args[0])
		os.Exit(-1)
		return
	}

	authUrl = os.Args[1]
	casServer = os.Args[2]
	serviceUrl = os.Args[3]
	certFile := os.Args[4] // "/etc/yunion/certs/nginx-full.crt"
	keyFile := os.Args[5]  // "/etc/yunion/certs/nginx.key"
	port := os.Args[6]     // 18443

	app := appsrv.NewApplication("casfe", 1, false)
	app.AddHandler("GET", "/", defaultPage)
	app.ListenAndServeTLS(fmt.Sprintf("0.0.0.0:%s", port), certFile, keyFile)
}
