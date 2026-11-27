// Copyright 2026 Yunion
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

package dashboardcodegen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/pkg/errors"
)

var (
	reFence = regexp.MustCompile("(?is)```(?:javascript|js)\\s*\\n(.*?)```")

	reMountFunc = regexp.MustCompile(`(?m)(?:^|\s)(?:export\s+(?:default\s+)?(?:async\s+)?function\s+mount\s*\(|(?:async\s+)?function\s+mount\s*\(|(?:const|let|var)\s+mount\s*=\s*(?:async\s*)?(?:function\s*)?\()`)

	// Forbidden patterns: conservative lexical scan (no full JS AST).
	forbiddenPatterns = []struct {
		name string
		re   *regexp.Regexp
	}{
		{"eval", regexp.MustCompile(`(?i)\beval\s*\(`)},
		{"new Function", regexp.MustCompile(`(?i)\bnew\s+Function\s*\(`)},
		{"dynamic import", regexp.MustCompile(`(?i)\bimport\s*\(`)},
		{"Worker", regexp.MustCompile(`(?i)\b(?:Shared)?Worker\s*\(`)},
		{"ServiceWorker", regexp.MustCompile(`(?i)\bnavigator\.serviceWorker\b`)},
		{"WebSocket", regexp.MustCompile(`(?i)\bWebSocket\s*\(`)},
		{"fetch", regexp.MustCompile(`(?i)\bfetch\s*\(`)},
		{"XMLHttpRequest", regexp.MustCompile(`(?i)\bXMLHttpRequest\b`)},
		{"axios", regexp.MustCompile(`(?i)\baxios\b`)},
		{"document.cookie", regexp.MustCompile(`(?i)\bdocument\.cookie\b`)},
		{"localStorage", regexp.MustCompile(`(?i)\blocalStorage\b`)},
		{"sessionStorage", regexp.MustCompile(`(?i)\bsessionStorage\b`)},
		{"indexedDB", regexp.MustCompile(`(?i)\bindexedDB\b`)},
		{"parent/top/opener", regexp.MustCompile(`(?i)(?:\bwindow\.(?:parent|top|opener)\b|\b(?:parent|top|opener)\.(?:location|document|postMessage)\b)`)},
		{"innerHTML", regexp.MustCompile(`(?i)\.innerHTML\b`)},
		{"outerHTML", regexp.MustCompile(`(?i)\.outerHTML\b`)},
		{"document.write", regexp.MustCompile(`(?i)\bdocument\.write(?:ln)?\s*\(`)},
		{"setTimeout string", regexp.MustCompile(`(?i)\bset(?:Timeout|Interval)\s*\(\s*['"]`)},
	}

	// DefaultAllowlistPrefixes: only GET under these path prefixes may be requested via ctx.request.
	// Align with dashboard overview widgets (fetchDashboardUsage / Quota / alerts).
	DefaultAllowlistPrefixes = []string{
		"/v2/rpc/usages/",
		"/v2/rpc/quotas",
		"/v2/rpc/region_quotas",
		"/v2/rpc/image_quotas",
		"/v2/usages/",
		"/v2/servers/",
		"/v1/servers",
		"/v1/hosts",
		"/v1/networks",
		"/v1/storages",
		"/v1/disks",
		"/v1/images",
		"/v1/vpcs",
		"/v1/cloudregions",
		"/v1/zones",
		"/v1/unifiedmonitors",
		"/v1/alertrecords",
		"/v1/suggestsysalerts",
		"/v1/notices",
		"/v1/bill_balances",
	}

	blockedPathPrefixes = []string{
		"/v1/auth",
		"/v1/credentials",
		"/v1/users",
		"/v1/projects",
		"/v1/domains",
		"/v1/roles",
		"/v1/policies",
		"/v1/tokens",
		"/v1/aksk",
		"/v1/access_keys",
		"/v1/secrets",
		"/v1/parameters",
		"/v1/usages", // legacy/wrong for overview; force /v2/rpc/usages/*
		"/v3/",
	}
)

// Result is the outcome of static validation.
type Result struct {
	Source        string
	Entry         string
	ContentSha256 string
	ApiAllowlist  []string
	Warnings      []string
}

// ExtractSource prefers fenced javascript blocks; otherwise returns trimmed full text.
func ExtractSource(modelText string) string {
	modelText = strings.TrimSpace(modelText)
	if modelText == "" {
		return ""
	}
	if m := reFence.FindStringSubmatch(modelText); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	// Strip a generic fence if present.
	if strings.HasPrefix(modelText, "```") {
		lines := strings.Split(modelText, "\n")
		if len(lines) >= 2 {
			end := len(lines)
			if strings.TrimSpace(lines[end-1]) == "```" {
				end--
			}
			return strings.TrimSpace(strings.Join(lines[1:end], "\n"))
		}
	}
	return modelText
}

// ContentSHA256 returns hex sha256 of source.
func ContentSHA256(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

// NormalizeScope returns system|domain|project, empty string, or error.
// Empty scope is allowed: frontend overview injects ctx.scope from the current view.
func NormalizeScope(scope string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(scope))
	if s == "" {
		return "", nil
	}
	switch s {
	case "system", "domain", "project":
		return s, nil
	default:
		return "", errors.Errorf("invalid scope %q (want system|domain|project, or omit)", scope)
	}
}

// ValidateSource performs static security checks and optional allowlist normalization.
func ValidateSource(source string, allowlist []string) (*Result, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, errors.Error("source is empty")
	}
	if len(source) > api.DashboardPanelMaxSourceBytes {
		return nil, errors.Errorf("source exceeds %d bytes", api.DashboardPanelMaxSourceBytes)
	}
	if !utf8.ValidString(source) {
		return nil, errors.Error("source is not valid UTF-8")
	}

	stripped := stripJSComments(source)
	for _, p := range forbiddenPatterns {
		if p.re.MatchString(stripped) {
			return nil, errors.Errorf("forbidden pattern: %s", p.name)
		}
	}
	if !reMountFunc.MatchString(source) {
		return nil, errors.Error("source must define function mount(root, ctx)")
	}

	normalized, warnings, err := NormalizeAllowlist(allowlist)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		normalized = defaultAllowlistEntries()
		warnings = append(warnings, "api_allowlist empty; using default read-only prefixes")
	}

	return &Result{
		Source:        source,
		Entry:         api.DashboardPanelEntryMount,
		ContentSha256: ContentSHA256(source),
		ApiAllowlist:  normalized,
		Warnings:      warnings,
	}, nil
}

func defaultAllowlistEntries() []string {
	out := make([]string, 0, len(DefaultAllowlistPrefixes))
	for _, p := range DefaultAllowlistPrefixes {
		out = append(out, "GET "+p)
	}
	return out
}

// NormalizeAllowlist validates and canonicalizes entries like "GET /v1/usages".
func NormalizeAllowlist(entries []string) ([]string, []string, error) {
	var warnings []string
	seen := make(map[string]struct{})
	out := make([]string, 0, len(entries))
	for _, raw := range entries {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		method, path, err := parseAllowEntry(raw)
		if err != nil {
			return nil, nil, err
		}
		if method != "GET" {
			return nil, nil, errors.Errorf("only GET is allowed in api_allowlist, got %s", method)
		}
		if err := assertAllowedPath(path); err != nil {
			return nil, nil, err
		}
		key := method + " " + path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out, warnings, nil
}

func parseAllowEntry(raw string) (method, path string, err error) {
	parts := strings.Fields(raw)
	if len(parts) == 1 {
		method = "GET"
		path = parts[0]
	} else if len(parts) >= 2 {
		method = strings.ToUpper(parts[0])
		path = parts[1]
	} else {
		return "", "", errors.Errorf("invalid api_allowlist entry %q", raw)
	}
	u, perr := url.Parse(path)
	if perr != nil {
		return "", "", errors.Wrapf(perr, "parse path %q", path)
	}
	if u.Scheme != "" || u.Host != "" {
		return "", "", errors.Errorf("api_allowlist path must be relative, got %q", path)
	}
	path = u.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return method, path, nil
}

func assertAllowedPath(path string) error {
	for _, b := range blockedPathPrefixes {
		if path == strings.TrimSuffix(b, "/") || strings.HasPrefix(path, b) {
			return errors.Errorf("path %q is blocked", path)
		}
	}
	ok := false
	for _, pref := range DefaultAllowlistPrefixes {
		if path == strings.TrimSuffix(pref, "/") || strings.HasPrefix(path, pref) {
			ok = true
			break
		}
	}
	if !ok {
		return errors.Errorf("path %q is outside allowed prefixes", path)
	}
	return nil
}

func stripJSComments(src string) string {
	// Remove /* ... */ then // line comments (best-effort; strings may false-positive — conservative).
	reBlock := regexp.MustCompile(`(?s)/\*.*?\*/`)
	s := reBlock.ReplaceAllString(src, " ")
	reLine := regexp.MustCompile(`(?m)//.*$`)
	return reLine.ReplaceAllString(s, " ")
}

// TruncateRunes truncates s to at most n runes.
func TruncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n])
}

// ValidatePromptTitle checks generate/save metadata limits.
func ValidatePromptTitle(prompt, title string) error {
	if strings.TrimSpace(prompt) == "" {
		return errors.Error("prompt is required")
	}
	if utf8.RuneCountInString(prompt) > api.DashboardPanelMaxPromptRunes {
		return errors.Errorf("prompt exceeds %d characters", api.DashboardPanelMaxPromptRunes)
	}
	if utf8.RuneCountInString(title) > api.DashboardPanelMaxTitleRunes {
		return errors.Errorf("title exceeds %d characters", api.DashboardPanelMaxTitleRunes)
	}
	return nil
}

// ScriptParameterName builds dashboard-custom-script-{uuid}.
func ScriptParameterName(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, api.DashboardPanelScriptNamePref) {
		return id
	}
	return api.DashboardPanelScriptNamePref + id
}

// FormatForbiddenHelp returns a short message for clients.
func FormatForbiddenHelp(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("script validation failed: %v", err)
}
