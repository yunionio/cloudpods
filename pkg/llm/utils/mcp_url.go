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

package utils

import (
	"net"
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/llm/options"
)

func ValidateMCPServerURL(raw string) error {
	u, err := parseMCPServerURL(raw)
	if err != nil {
		return err
	}
	if IsTrustedMCPServerURL(raw) {
		return nil
	}
	return rejectRestrictedMCPHost(u.Hostname())
}

func IsTrustedMCPServerURL(raw string) bool {
	configured := strings.TrimSpace(options.Options.MCPServerURL)
	if configured == "" {
		return false
	}
	got, err := parseMCPServerURL(raw)
	if err != nil {
		return false
	}
	want, err := parseMCPServerURL(configured)
	if err != nil {
		return false
	}
	return mcpOrigin(got) == mcpOrigin(want)
}

func parseMCPServerURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.Errorf("invalid mcp server url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid mcp server url")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, errors.Errorf("invalid mcp server url")
	}
	if u.Host == "" || u.Opaque != "" || u.User != nil || u.Fragment != "" || strings.Contains(raw, "#") {
		return nil, errors.Errorf("invalid mcp server url")
	}
	if u.Hostname() == "" {
		return nil, errors.Errorf("invalid mcp server url")
	}
	return u, nil
}

func rejectRestrictedMCPHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return errors.Errorf("invalid mcp server url")
	}
	ips := []net.IP{}
	if ip := net.ParseIP(host); ip != nil {
		ips = append(ips, ip)
	} else {
		resolved, err := net.LookupIP(host)
		if err != nil || len(resolved) == 0 {
			return errors.Errorf("invalid mcp server url")
		}
		ips = resolved
	}
	for _, ip := range ips {
		if isRestrictedMCPIP(ip) {
			return errors.Errorf("invalid mcp server url")
		}
	}
	return nil
}

func isRestrictedMCPIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}

func mcpOrigin(u *url.URL) string {
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return scheme + "://" + host + ":" + port
}

func joinMCPEndpoint(serverURL, endpoint string) (string, error) {
	base, err := parseMCPServerURL(serverURL)
	if err != nil {
		return "", err
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", errors.Errorf("invalid mcp endpoint")
	}
	rel, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "invalid mcp endpoint")
	}
	if strings.HasPrefix(endpoint, "//") || rel.Opaque != "" || rel.User != nil || rel.Fragment != "" {
		return "", errors.Errorf("invalid mcp endpoint")
	}
	var joined *url.URL
	if rel.IsAbs() || rel.Host != "" {
		joined = rel
	} else {
		joined = base.ResolveReference(rel)
	}
	if mcpOrigin(base) != mcpOrigin(joined) {
		return "", errors.Errorf("invalid mcp endpoint")
	}
	return joined.String(), nil
}
