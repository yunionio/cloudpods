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

package handlers

import (
	"context"
	"net/http"
)

const anthropicBasePrefix = "/ai/anthropic"

// anthropicBaseProbeHandler answers HEAD on the Anthropic base URL (e.g. /ai/anthropic or /ai/anthropic/).
// appsrv SplitPath normalizes trailing slashes; one route covers both. Claude / Anthropic SDK probes
// base URL connectivity before POST /v1/messages.
func anthropicBaseProbeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// anthropicMessagesHeadHandler answers HEAD on /ai/anthropic/v1/messages for path-existence probes.
// Returns 401 without virtual key (route exists, auth required); 204 when a key is present.
func anthropicMessagesHeadHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if extractVirtualKey(r) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
