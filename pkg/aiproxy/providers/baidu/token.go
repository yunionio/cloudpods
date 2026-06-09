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

package baidu

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultTokenURL = "https://aip.baidubce.com/oauth/2.0/token"
	tokenSkew       = 5 * time.Minute
)

type cachedToken struct {
	value  string
	expiry time.Time
}

var tokenCache sync.Map

// ResolveAccessToken returns a Wenxin access_token.
// apiKey may be a raw access_token, or "APIKey:SecretKey" for OAuth exchange.
func ResolveAccessToken(apiKey string) (string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", fmt.Errorf("empty baidu api key")
	}
	if !strings.Contains(apiKey, ":") {
		return apiKey, nil
	}
	parts := strings.SplitN(apiKey, ":", 2)
	clientID := strings.TrimSpace(parts[0])
	clientSecret := strings.TrimSpace(parts[1])
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("invalid baidu api key format, want access_token or APIKey:SecretKey")
	}
	cacheKey := clientID + ":" + clientSecret
	if v, ok := tokenCache.Load(cacheKey); ok {
		entry := v.(cachedToken)
		if time.Now().Before(entry.expiry.Add(-tokenSkew)) {
			return entry.value, nil
		}
	}
	token, expiresIn, err := fetchAccessToken(clientID, clientSecret)
	if err != nil {
		return "", err
	}
	if expiresIn <= 0 {
		expiresIn = 30 * 24 * time.Hour
	}
	tokenCache.Store(cacheKey, cachedToken{
		value:  token,
		expiry: time.Now().Add(expiresIn),
	})
	return token, nil
}

func fetchAccessToken(clientID, clientSecret string) (string, time.Duration, error) {
	q := url.Values{}
	q.Set("grant_type", "client_credentials")
	q.Set("client_id", clientID)
	q.Set("client_secret", clientSecret)
	req, err := http.NewRequest(http.MethodPost, defaultTokenURL+"?"+q.Encode(), nil)
	if err != nil {
		return "", 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, fmt.Errorf("baidu oauth HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var wrap struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return "", 0, err
	}
	if wrap.AccessToken == "" {
		msg := wrap.ErrorDesc
		if msg == "" {
			msg = wrap.Error
		}
		if msg == "" {
			msg = string(body)
		}
		return "", 0, fmt.Errorf("baidu oauth failed: %s", msg)
	}
	return wrap.AccessToken, time.Duration(wrap.ExpiresIn) * time.Second, nil
}

func wenxinBaseURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "https://aip.baidubce.com"
	}
	return base
}

func wenxinChatURL(baseURL, model, accessToken string) string {
	path := fmt.Sprintf("/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/%s", strings.TrimSpace(model))
	u := wenxinBaseURL(baseURL) + path
	return u + "?access_token=" + url.QueryEscape(accessToken)
}

func wenxinEmbeddingsURL(baseURL, model, accessToken string) string {
	path := fmt.Sprintf("/rpc/2.0/ai_custom/v1/wenxinworkshop/embeddings/%s", strings.TrimSpace(model))
	u := wenxinBaseURL(baseURL) + path
	return u + "?access_token=" + url.QueryEscape(accessToken)
}

func useQianfanV2(baseURL string) bool {
	base := strings.ToLower(strings.TrimSpace(baseURL))
	if base == "" {
		return true
	}
	return strings.Contains(base, "qianfan.baidubce.com")
}
