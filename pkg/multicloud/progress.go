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

package multicloud

import (
	"io"
	"time"
)

func NewProgress(totalSize int64, maxPercent int, reader io.Reader, callback func(progress float32)) io.Reader {
	body := &sProgress{
		total:      totalSize,
		maxPercent: maxPercent,
		callback:   callback,
	}
	return io.TeeReader(reader, body)
}

type sProgress struct {
	refreshSeconds int
	count          int64
	total          int64
	start          time.Time
	callback       func(progress float32)
	maxPercent     int
}

func (r *sProgress) Write(p []byte) (int, error) {
	if r.start.IsZero() {
		r.start = time.Now()
	}
	n := len(p)
	r.count += int64(n)
	if r.callback != nil && r.total > 0 && time.Now().Sub(r.start) > time.Second*1 {
		r.callback(float32(float64(r.count) / float64(r.total) * float64(r.maxPercent)))
		r.start = time.Now()
	}
	return n, nil
}
