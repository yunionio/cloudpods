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

// EnforceVirtualKeyMaxTokens caps or injects max_tokens from virtual key limits.
func EnforceVirtualKeyMaxTokens(body *jsonutils.JSONDict, lim *api.SAiVirtualKeyLimits) error {
	if lim == nil || lim.MaxTokensPerRequest <= 0 {
		return nil
	}
	cap := int64(lim.MaxTokensPerRequest)
	if body.Contains("max_tokens") {
		mt, err := body.Int("max_tokens")
		if err != nil {
			return errors.Wrap(httperrors.ErrInputParameter, "invalid max_tokens")
		}
		if mt > cap {
			return errors.Wrap(httperrors.ErrInputParameter, "max_tokens exceeds virtual key limit")
		}
		return nil
	}
	body.Set("max_tokens", jsonutils.NewInt(cap))
	return nil
}
