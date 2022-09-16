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

package notifiers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/context/ctxhttp"
	"moul.io/http2curl/v2"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

// GetBasicAuthHeader returns a base64 encoded string from user and password.
func GetBasicAuthHeader(user string, password string) string {
	var userAndPass = user + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(userAndPass))
}

// DecodeBasicAuthHeader decodes user and password from a basic auth header.
func DecodeBasicAuthHeader(header string) (string, string, error) {
	var code string
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && parts[0] == "Basic" {
		code = parts[1]
	}

	decoded, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		return "", "", err
	}

	userAndPass := strings.SplitN(string(decoded), ":", 2)
	if len(userAndPass) != 2 {
		return "", "", fmt.Errorf("Invalid basic auth header")
	}

	return userAndPass[0], userAndPass[1], nil
}

// TODO: use httputils.HttpTransport instead -- qj
// TODO:
var netTransport = &http.Transport{
	TLSClientConfig: &tls.Config{
		Renegotiation: tls.RenegotiateFreelyAsClient,
	},
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout: 30 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout: 5 * time.Second,
}
var netClient = &http.Client{
	Timeout:   time.Second * 30,
	Transport: netTransport,
}

func SendWebRequestSync(ctx context.Context, webhook *monitor.SendWebhookSync) error {
	if webhook.HttpMethod == "" {
		webhook.HttpMethod = http.MethodPost
	}

	request, err := http.NewRequest(webhook.HttpMethod, webhook.Url, bytes.NewReader([]byte(webhook.Body)))
	if err != nil {
		return err
	}

	if webhook.ContentType == "" {
		webhook.ContentType = "application/json"
	}

	request.Header.Add("Content-Type", webhook.ContentType)
	request.Header.Add("User-Agent", "OneCloud Monitor")

	if webhook.User != "" && webhook.Password != "" {
		request.Header.Add("Authorization", GetBasicAuthHeader(webhook.User, webhook.Password))
	}

	for k, v := range webhook.HttpHeader {
		request.Header.Set(k, v)
	}

	curlCmd, _ := http2curl.GetCurlCommand(request)
	log.Debugf("webhook curl: %s", curlCmd)

	resp, err := ctxhttp.Do(ctx, netClient, request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode/100 == 2 {
		// flushing the body enables the transport to reuse the same connection
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			log.Errorf("Failed to copy resp.Body to io.Discard: %v", err)
		}
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Errorf("Webhook failed statuscode: %s, body: %s", resp.Status, string(body))
	return fmt.Errorf("Webhook response status %v", resp.Status)
}
