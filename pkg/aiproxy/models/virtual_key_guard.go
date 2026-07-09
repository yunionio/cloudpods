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

package models

import (
	"strings"
	"sync"

	"golang.org/x/time/rate"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/httperrors"
)

var vkRpmLimiters sync.Map // virtual key id -> *rate.Limiter

// TakeVirtualKeyRequestsPerMinute enforces an approximate per-minute request budget per virtual key (in-process).
func TakeVirtualKeyRequestsPerMinute(vkId string, rpm int) error {
	if rpm <= 0 || strings.TrimSpace(vkId) == "" {
		return nil
	}
	limAny, _ := vkRpmLimiters.LoadOrStore(vkId, rate.NewLimiter(rate.Limit(float64(rpm))/60.0, rpm))
	lim := limAny.(*rate.Limiter)
	if !lim.Allow() {
		return errors.Wrap(httperrors.ErrTooManyRequests, "virtual key request rate exceeded")
	}
	return nil
}

// EnsureResponsesMaxOutputTokens caps or injects max_output_tokens for Responses API requests.
func EnsureResponsesMaxOutputTokens(body *jsonutils.JSONDict, lim *api.SAiVirtualKeyLimits) error {
	if err := EnforceVirtualKeyMaxTokens(body, lim); err != nil {
		return err
	}
	if body.Contains("max_output_tokens") || body.Contains("max_tokens") {
		return nil
	}
	body.Set("max_output_tokens", jsonutils.NewInt(defaultResponsesMaxOutputTokens))
	return nil
}

const defaultResponsesMaxOutputTokens = 8192

// EnforceVirtualKeyMaxTokens caps or injects max_tokens / max_output_tokens from virtual key limits.
func EnforceVirtualKeyMaxTokens(body *jsonutils.JSONDict, lim *api.SAiVirtualKeyLimits) error {
	if lim == nil || lim.MaxTokensPerRequest <= 0 {
		return nil
	}
	cap := int64(lim.MaxTokensPerRequest)
	for _, field := range []string{"max_tokens", "max_output_tokens"} {
		if body.Contains(field) {
			mt, err := body.Int(field)
			if err != nil {
				return errors.Wrap(httperrors.ErrInputParameter, "invalid "+field)
			}
			if mt > cap {
				return errors.Wrap(httperrors.ErrInputParameter, field+" exceeds virtual key limit")
			}
			continue
		}
	}
	if body.Contains("max_tokens") || body.Contains("max_output_tokens") {
		return nil
	}
	body.Set("max_output_tokens", jsonutils.NewInt(cap))
	return nil
}
