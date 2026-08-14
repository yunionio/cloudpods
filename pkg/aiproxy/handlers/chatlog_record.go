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
	"encoding/json"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/appctx"

	"yunion.io/x/onecloud/pkg/aiproxy/chatlog"
	"yunion.io/x/onecloud/pkg/aiproxy/models"
)

func newAPILogRecord(ctx context.Context, r *http.Request, start time.Time) *chatlog.Record {
	return &chatlog.Record{
		RequestID: appctx.AppContextRequestId(ctx),
		Timestamp: start,
		Path:      r.URL.Path,
		Client:    r.RemoteAddr,
	}
}

func finishAPILogRecord(rec *chatlog.Record, start time.Time) {
	if rec == nil {
		return
	}
	if rec.StatusCode == 0 {
		rec.StatusCode = http.StatusInternalServerError
	}
	rec.LatencyMs = time.Since(start).Milliseconds()
	chatlog.Write(rec)
}

func failAPILogRecord(rec *chatlog.Record, status int, code string, err error) {
	if rec == nil {
		return
	}
	rec.StatusCode = status
	rec.Success = false
	rec.ErrorCode = code
	if err != nil {
		rec.ErrorMessage = err.Error()
	}
}

func fillAPILogFromBody(rec *chatlog.Record, dict *jsonutils.JSONDict) {
	if rec == nil || dict == nil {
		return
	}
	rec.ModelRequested, _ = dict.GetString("model")
	rec.Stream, _ = dict.Bool("stream")
	rec.ToolCallEnabled = dict.Contains("tools") || dict.Contains("tool_choice")
	if metadata, err := dict.Get("metadata"); err == nil {
		rec.Metadata = json.RawMessage([]byte(metadata.String()))
	}
}

func fillAPILogFromUpstream(rec *chatlog.Record, up *models.ChatUpstream) {
	if rec == nil || up == nil {
		return
	}
	rec.VirtualKey = up.VirtualKeyId
	rec.ProjectID = up.ProjectId
	rec.DomainID = up.DomainId
	rec.AiKey = up.AiKeyId
	rec.ModelFinal = up.UpstreamModel
	rec.Provider = up.ProviderKey
	rec.AiProviderId = up.AiProviderId
	if up.RoutingLog != nil {
		rec.RoutingEnabled = up.RoutingLog.Enabled
		rec.RoutingCandidates = up.RoutingLog.Candidates
		rec.RoutingSelectedModel = up.RoutingLog.SelectedModel
		rec.RoutingMethod = up.RoutingLog.Method
		rec.RoutingScores = up.RoutingLog.Scores
		rec.RoutingConfidence = up.RoutingLog.Confidence
		rec.RoutingReason = up.RoutingLog.Reason
		rec.RoutingLatencyMs = up.RoutingLog.LatencyMs
		rec.RoutingError = up.RoutingLog.Error
	}
}

func markAPILogSuccess(rec *chatlog.Record, body []byte) {
	if rec == nil {
		return
	}
	if len(body) > 0 {
		chatlog.FillUsageFromJSON(rec, body)
	} else {
		rec.UsageMissing = true
	}
	rec.Success = true
	rec.StatusCode = http.StatusOK
}

func finishAPILogStream(rec *chatlog.Record, streamOK, sawUsage bool) {
	if rec == nil {
		return
	}
	rec.StatusCode = http.StatusOK
	if sawUsage {
		rec.UsageMissing = false
	} else {
		rec.UsageMissing = true
	}
	if streamOK {
		rec.Success = true
	}
}

func noteStreamUsage(rec *chatlog.Record, data []byte, sawUsage *bool) {
	if rec == nil || len(data) == 0 || sawUsage == nil {
		return
	}
	if chatlog.FillUsageFromJSON(rec, data) {
		*sawUsage = true
	}
}
