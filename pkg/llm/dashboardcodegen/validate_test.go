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
	"strings"
	"testing"
)

func TestExtractSourceFence(t *testing.T) {
	raw := "说明\n```javascript\nfunction mount(root, ctx) {}\n```\n"
	got := ExtractSource(raw)
	if !strings.Contains(got, "function mount") {
		t.Fatalf("unexpected extract: %q", got)
	}
}

func TestValidateSourceOK(t *testing.T) {
	src := `
function mount(root, ctx) {
  const el = document.createElement('div');
  el.textContent = 'ok';
  root.appendChild(el);
  ctx.request('GET', '/v2/rpc/usages/general-usage', { scope: ctx.scope, refresh: true }).then(function () {});
}
`
	res, err := ValidateSource(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry != "mount" {
		t.Fatalf("entry=%s", res.Entry)
	}
	if res.ContentSha256 == "" {
		t.Fatal("missing sha256")
	}
	if len(res.ApiAllowlist) == 0 {
		t.Fatal("expected default allowlist")
	}
	found := false
	for _, e := range res.ApiAllowlist {
		if strings.Contains(e, "/v2/rpc/usages/") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("default allowlist missing usages rpc: %#v", res.ApiAllowlist)
	}
}

func TestValidateSourceForbidden(t *testing.T) {
	cases := []string{
		`function mount(root, ctx) { eval('1'); }`,
		`function mount(root, ctx) { fetch('/v2/rpc/usages/general-usage'); }`,
		`function mount(root, ctx) { root.innerHTML = '<b>x</b>'; }`,
		`function mount(root, ctx) { localStorage.setItem('a','b'); }`,
		`const x = 1;`, // no mount
	}
	for _, src := range cases {
		if _, err := ValidateSource(src, nil); err == nil {
			t.Fatalf("expected error for %q", src)
		}
	}
}

func TestNormalizeAllowlist(t *testing.T) {
	out, _, err := NormalizeAllowlist([]string{"GET /v2/rpc/usages/general-usage", "/v1/servers"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("got %#v", out)
	}
	if _, _, err := NormalizeAllowlist([]string{"POST /v1/servers"}); err == nil {
		t.Fatal("expected POST rejected")
	}
	if _, _, err := NormalizeAllowlist([]string{"GET /v1/users"}); err == nil {
		t.Fatal("expected blocked path")
	}
	if _, _, err := NormalizeAllowlist([]string{"GET /v1/usages"}); err == nil {
		t.Fatal("expected legacy /v1/usages blocked")
	}
}

func TestExportMountAccepted(t *testing.T) {
	src := `export function mount(root, ctx) { const d = document.createElement('div'); root.appendChild(d); }`
	if _, err := ValidateSource(src, []string{"GET /v2/rpc/usages/general-usage"}); err != nil {
		t.Fatal(err)
	}
}
