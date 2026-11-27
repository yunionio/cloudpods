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
	"sync"
	"time"

	"yunion.io/x/pkg/errors"
)

const (
	generateMinInterval = 3 * time.Second
	generateMaxInflight = 1
)

type userLimiter struct {
	mu       sync.Mutex
	last     time.Time
	inflight int
}

var (
	limitMu sync.Mutex
	limits  = map[string]*userLimiter{}
)

func getLimiter(userId string) *userLimiter {
	limitMu.Lock()
	defer limitMu.Unlock()
	l, ok := limits[userId]
	if !ok {
		l = &userLimiter{}
		limits[userId] = l
	}
	return l
}

// AcquireGenerate reserves a generate slot for userId. Caller must ReleaseGenerate.
func AcquireGenerate(userId string) error {
	if userId == "" {
		userId = "_"
	}
	l := getLimiter(userId)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inflight >= generateMaxInflight {
		return errors.Error("dashboard panel generate already in progress for this user")
	}
	if !l.last.IsZero() && time.Since(l.last) < generateMinInterval {
		return errors.Error("dashboard panel generate rate limited; retry shortly")
	}
	l.inflight++
	return nil
}

// ReleaseGenerate marks the generate call finished.
func ReleaseGenerate(userId string) {
	if userId == "" {
		userId = "_"
	}
	l := getLimiter(userId)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inflight > 0 {
		l.inflight--
	}
	l.last = time.Now()
}
