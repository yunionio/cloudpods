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
	"time"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/upstream"
)

type upstreamRequestBuilder func() (*upstream.Request, error)

func upstreamWithKeyFailover(
	ctx context.Context,
	up *models.ChatUpstream,
	timeout time.Duration,
	build upstreamRequestBuilder,
) (*upstream.Response, *upstream.Error) {
	tried := make(map[string]bool)
	if up.AiKeyId != "" {
		tried[up.AiKeyId] = true
	}
	var last *upstream.Error
	for attempt := 0; attempt < models.MaxAiKeyFailoverAttempts; attempt++ {
		upReq, err := build()
		if err != nil {
			return nil, &upstream.Error{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		resp, uerr := upstream.ChatCompletion(reqCtx, upReq)
		cancel()
		if uerr == nil {
			models.RecordAiKeySuccess(up.AiKeyId)
			return resp, nil
		}
		last = uerr
		status := upstreamErrorStatusCode(uerr)
		models.RecordAiKeyFailure(up.AiKeyId, status)
		if up.AiKeyId == "" || !models.IsRetryableAiKeyUpstreamStatus(status) || attempt+1 >= models.MaxAiKeyFailoverAttempts {
			break
		}
		if err := models.RepickUpstreamAPIKey(up, tried); err != nil {
			break
		}
		if up.AiKeyId != "" {
			tried[up.AiKeyId] = true
		}
	}
	return nil, last
}

type streamChunkProducer func(ctx context.Context, upReq *upstream.Request) (<-chan upstream.StreamChunk, *upstream.Error)

func upstreamStreamWithKeyFailover(
	ctx context.Context,
	up *models.ChatUpstream,
	timeout time.Duration,
	build upstreamRequestBuilder,
	produce streamChunkProducer,
) (<-chan upstream.StreamChunk, *upstream.Error) {
	tried := make(map[string]bool)
	if up.AiKeyId != "" {
		tried[up.AiKeyId] = true
	}
	var last *upstream.Error
	for attempt := 0; attempt < models.MaxAiKeyFailoverAttempts; attempt++ {
		upReq, err := build()
		if err != nil {
			return nil, &upstream.Error{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		ch, uerr := produce(reqCtx, upReq)
		if uerr != nil {
			cancel()
		} else {
			ch = streamChunksWithCancel(ch, cancel)
		}
		if uerr == nil {
			return ch, nil
		}
		last = uerr
		status := upstreamErrorStatusCode(uerr)
		models.RecordAiKeyFailure(up.AiKeyId, status)
		if up.AiKeyId == "" || !models.IsRetryableAiKeyUpstreamStatus(status) || attempt+1 >= models.MaxAiKeyFailoverAttempts {
			break
		}
		if err := models.RepickUpstreamAPIKey(up, tried); err != nil {
			break
		}
		if up.AiKeyId != "" {
			tried[up.AiKeyId] = true
		}
	}
	return nil, last
}

type rawSSEProducer func(ctx context.Context, upReq *upstream.Request) (<-chan upstream.RawSSEEvent, *upstream.Error)

func upstreamRawStreamWithKeyFailover(
	ctx context.Context,
	up *models.ChatUpstream,
	timeout time.Duration,
	build upstreamRequestBuilder,
	produce rawSSEProducer,
) (<-chan upstream.RawSSEEvent, *upstream.Error) {
	tried := make(map[string]bool)
	if up.AiKeyId != "" {
		tried[up.AiKeyId] = true
	}
	var last *upstream.Error
	for attempt := 0; attempt < models.MaxAiKeyFailoverAttempts; attempt++ {
		upReq, err := build()
		if err != nil {
			return nil, &upstream.Error{StatusCode: http.StatusBadRequest, Message: err.Error()}
		}
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		ch, uerr := produce(reqCtx, upReq)
		if uerr != nil {
			cancel()
		} else {
			ch = rawSSEWithCancel(ch, cancel)
		}
		if uerr == nil {
			return ch, nil
		}
		last = uerr
		status := upstreamErrorStatusCode(uerr)
		models.RecordAiKeyFailure(up.AiKeyId, status)
		if up.AiKeyId == "" || !models.IsRetryableAiKeyUpstreamStatus(status) || attempt+1 >= models.MaxAiKeyFailoverAttempts {
			break
		}
		if err := models.RepickUpstreamAPIKey(up, tried); err != nil {
			break
		}
		if up.AiKeyId != "" {
			tried[up.AiKeyId] = true
		}
	}
	return nil, last
}

func rawSSEWithCancel(ch <-chan upstream.RawSSEEvent, cancel context.CancelFunc) <-chan upstream.RawSSEEvent {
	out := make(chan upstream.RawSSEEvent, 16)
	go func() {
		defer cancel()
		defer close(out)
		for evt := range ch {
			out <- evt
		}
	}()
	return out
}
