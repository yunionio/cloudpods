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

package azure

import (
	"strings"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestResponsesURLAPIVersion(t *testing.T) {
	body := jsonutils.NewDict()
	body.Set("api-version", jsonutils.NewString("2024-12-01-preview"))
	url, err := ResponsesURL("https://example.openai.azure.com", body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "api-version=2024-12-01-preview") {
		t.Fatalf("url = %q", url)
	}
	if !strings.Contains(url, "/openai/responses") {
		t.Fatalf("url = %q", url)
	}
}

func TestResponsesSubResourceURL(t *testing.T) {
	body := jsonutils.NewDict()
	body.Set("api-version", jsonutils.NewString("2024-10-01"))
	url, err := ResponsesSubResourceURL("https://example.openai.azure.com", "resp_abc", "cancel", body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "/openai/responses/resp_abc/cancel") {
		t.Fatalf("url = %q", url)
	}
	if !strings.Contains(url, "api-version=2024-10-01") {
		t.Fatalf("url = %q", url)
	}
}
