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

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
)

// bearerTokenResp is the json response of a Docker Registry token service,
// same shape that docker login consumes.
type bearerTokenResp struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
}

var authParamRegexp = regexp.MustCompile(`(?i)([a-z]+)=(?:"([^"]*)"|([^\s,]+))`)

// parseWWWAuthenticate parses a single WWW-Authenticate header value into
// scheme (basic/bearer) and parameter map (realm, service, scope, ...).
func parseWWWAuthenticate(header string) (scheme string, params map[string]string, err error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", nil, errors.Error("empty WWW-Authenticate header")
	}
	parts := strings.SplitN(header, " ", 2)
	scheme = strings.ToLower(parts[0])
	params = map[string]string{}
	if len(parts) < 2 {
		return scheme, params, nil
	}
	for _, m := range authParamRegexp.FindAllStringSubmatch(parts[1], -1) {
		key := strings.ToLower(m[1])
		if m[2] != "" {
			params[key] = m[2]
		} else {
			params[key] = m[3]
		}
	}
	return scheme, params, nil
}

// LoginRegistry validates username/password against a Docker Registry using the
// same authentication challenge flow as `docker login` (Registry HTTP API V2):
//  1. GET /v2/ without Authorization
//  2. On 401, honor WWW-Authenticate (Bearer token exchange or Basic retry)
//
// It does not require a local docker daemon.
func LoginRegistry(ctx context.Context, regUrl string, authConf DockerAuthConfig) error {
	if authConf.Username == "" || authConf.Password == "" {
		return errors.Error("username and password are required for registry login")
	}
	u, err := url.Parse(regUrl)
	if err != nil {
		return errors.Wrapf(err, "parse registry url %s", regUrl)
	}
	baseUrl := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

	cli := httputils.GetDefaultClient()
	pingUrl := joinUrl(baseUrl, "/v2/")
	resp, err := httputils.Request(cli, ctx, httputils.GET, pingUrl, nil, nil, false)
	if err != nil {
		return errors.Wrapf(err, "ping registry %q", baseUrl)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		// Anonymous access allowed; credentials cannot be further verified via
		// /v2/ challenge. Treat as connectivity success (same as docker login
		// against an open registry).
		return nil
	case http.StatusUnauthorized:
		return loginFromChallenge(ctx, cli, baseUrl, authConf, resp.Header)
	default:
		return errors.Errorf("unexpected status %d from registry %q", resp.StatusCode, baseUrl)
	}
}

func loginFromChallenge(ctx context.Context, cli *http.Client, baseUrl string, authConf DockerAuthConfig, header http.Header) error {
	wwwAuth := header.Values("WWW-Authenticate")
	if len(wwwAuth) == 0 {
		// Some registries return 401 without a challenge; fall back to Basic.
		return loginWithBasic(ctx, cli, baseUrl, authConf)
	}
	var lastErr error
	for _, h := range wwwAuth {
		scheme, params, err := parseWWWAuthenticate(h)
		if err != nil {
			lastErr = err
			continue
		}
		switch scheme {
		case "bearer":
			if err := loginWithBearer(ctx, cli, authConf, params); err != nil {
				return errors.Wrap(err, "bearer token login")
			}
			return nil
		case "basic":
			if err := loginWithBasic(ctx, cli, baseUrl, authConf); err != nil {
				return errors.Wrap(err, "basic auth login")
			}
			return nil
		default:
			lastErr = errors.Errorf("unsupported auth scheme %q", scheme)
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.Error("no usable WWW-Authenticate challenge")
}

func loginWithBasic(ctx context.Context, cli *http.Client, baseUrl string, authConf DockerAuthConfig) error {
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Basic %s", basicAuth(authConf.Username, authConf.Password)))
	pingUrl := joinUrl(baseUrl, "/v2/")
	resp, err := httputils.Request(cli, ctx, httputils.GET, pingUrl, header, nil, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return errors.Error("invalid registry username or password")
	}
	return errors.Errorf("basic auth login unexpected status %d", resp.StatusCode)
}

func loginWithBearer(ctx context.Context, cli *http.Client, authConf DockerAuthConfig, params map[string]string) error {
	realm := params["realm"]
	if realm == "" {
		return errors.Error("bearer challenge missing realm")
	}
	service := params["service"]
	scope := params["scope"]
	if scope == "" {
		// docker login against /v2/ often gets a challenge without repository
		// scope; auth servers still accept an empty or registry-level request.
		scope = ""
	}

	// Prefer GET (docker login / distribution default), fall back to POST.
	if err := bearerTokenGet(ctx, cli, realm, service, scope, authConf); err != nil {
		if err2 := bearerTokenPost(ctx, cli, realm, service, scope, authConf); err2 != nil {
			return errors.Wrapf(err, "GET token failed, POST also failed: %v", err2)
		}
	}
	return nil
}

func bearerTokenGet(ctx context.Context, cli *http.Client, realm, service, scope string, authConf DockerAuthConfig) error {
	u, err := url.Parse(realm)
	if err != nil {
		return errors.Wrap(err, "parse bearer realm")
	}
	q := u.Query()
	q.Set("account", authConf.Username)
	if service != "" {
		q.Set("service", service)
	}
	if scope != "" {
		q.Set("scope", scope)
	}
	u.RawQuery = q.Encode()

	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Basic %s", basicAuth(authConf.Username, authConf.Password)))
	resp, err := httputils.Request(cli, ctx, httputils.GET, u.String(), header, nil, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return validateBearerTokenResponse(resp)
}

func bearerTokenPost(ctx context.Context, cli *http.Client, realm, service, scope string, authConf DockerAuthConfig) error {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", authConf.Username)
	form.Set("password", authConf.Password)
	form.Set("client_id", "yunion-container-registry")
	if service != "" {
		form.Set("service", service)
	}
	if scope != "" {
		form.Set("scope", scope)
	}
	header := http.Header{}
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httputils.Request(cli, ctx, httputils.POST, realm, header, strings.NewReader(form.Encode()), false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return validateBearerTokenResponse(resp)
}

func validateBearerTokenResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read token response")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return errors.Error("invalid registry username or password")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}
	var tok bearerTokenResp
	if err := json.Unmarshal(body, &tok); err != nil {
		return errors.Wrap(err, "decode token response")
	}
	if tok.Token == "" && tok.AccessToken == "" {
		return errors.Error("token endpoint returned empty token")
	}
	return nil
}

// Login validates credentials with the docker-login authentication challenge flow.
func (c client) Login(ctx context.Context) error {
	return LoginRegistry(ctx, c.regUrl, c.authConfig)
}
