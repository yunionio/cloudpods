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

package huawei

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"
)

// ConvertNovncHtmlUrlToWebsocket converts a novnc html console url to a
// websockify websocket url for webconsole wsproxy.
//
// Huawei public cloud example:
//
//	https://console-intl.huaweicloud.com:443/vnc/region.ap-southeast-1/vnc_auto.html?token=UUID&lang=EN
//	-> wss://console-intl.huaweicloud.com:443/vnc/region.ap-southeast-1/websockify?token=UUID
//
// Classic openstack/hcso example:
//
//	https://host:8002/vnc_auto.html?token=UUID&lang=EN
//	-> wss://host:8002/websockify?token=UUID
func ConvertNovncHtmlUrlToWebsocket(htmlUrl string) (string, error) {
	u, err := url.Parse(htmlUrl)
	if err != nil {
		return "", errors.Wrapf(err, "parse vnc url %s", htmlUrl)
	}
	if len(u.Host) == 0 {
		return "", errors.Errorf("vnc url missing host: %s", htmlUrl)
	}

	token := u.Query().Get("token")
	path := u.Query().Get("path")
	if len(path) == 0 {
		path = "websockify"
	}
	path = strings.TrimPrefix(path, "/")

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	// path may already embed token, e.g. "websockify?token=xxx" or "?token=xxx"
	if strings.Contains(path, "token=") {
		wsPath := strings.TrimPrefix(path, "?")
		// keep directory prefix from html path when relative
		if prefix := novncDirPrefix(u.Path); len(prefix) > 0 && !strings.HasPrefix(wsPath, "/") && !strings.Contains(wsPath, "/") {
			wsPath = strings.TrimSuffix(prefix, "/") + "/" + wsPath
		}
		return fmt.Sprintf("%s://%s/%s", scheme, u.Host, strings.TrimPrefix(wsPath, "/")), nil
	}
	if len(token) == 0 {
		return "", errors.Errorf("vnc url missing token: %s", htmlUrl)
	}

	// Huawei/noVNC resolves websockify relative to the html directory:
	// pathname ".../vnc_auto.html" -> prefix "...", websocket at "{prefix}/websockify".
	prefix := novncDirPrefix(u.Path)
	wsPath := path
	if len(prefix) > 0 {
		wsPath = strings.TrimSuffix(prefix, "/") + "/" + path
	}
	return fmt.Sprintf("%s://%s/%s?token=%s", scheme, u.Host, strings.TrimPrefix(wsPath, "/"), token), nil
}

func novncDirPrefix(htmlPath string) string {
	for _, marker := range []string{"/vnc_auto.html", "/vnc_lite.html", "/vnc.html"} {
		if idx := strings.Index(htmlPath, marker); idx >= 0 {
			return htmlPath[:idx]
		}
	}
	if idx := strings.LastIndex(htmlPath, "/"); idx > 0 {
		return htmlPath[:idx]
	}
	return ""
}
