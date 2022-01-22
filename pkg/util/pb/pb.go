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

package pb

import (
	"context"
	"io"
	"time"

	"golang.org/x/time/rate"
)

const (
	BURSTS = 1024 * 1024 * 1024
)

type SProxyReader struct {
	reader  io.Reader
	size    int64
	cur     int64
	lastCur int64

	finish chan bool

	debug bool

	ticker *time.Ticker

	start    time.Time
	callback func()

	ctx context.Context

	refreshRate time.Duration

	// Mbps
	rateLimit *rate.Limiter
}

func NewProxyReader(reader io.Reader, size int64) *SProxyReader {
	return &SProxyReader{
		reader:      reader,
		size:        size,
		finish:      make(chan bool),
		ctx:         context.Background(),
		refreshRate: time.Second * 1,
	}
}

func (self *SProxyReader) SetRateLimit(mb int) {
	self.rateLimit = rate.NewLimiter(rate.Limit(mb*1024*1024), BURSTS)
	self.rateLimit.AllowN(time.Now(), BURSTS)
}

// 设置刷新频率
// 仅在读取数据前生效
func (self *SProxyReader) SetRefreshRate(rate time.Duration) {
	if rate > time.Second && self.cur == 0 {
		self.refreshRate = rate
	}
}

func (self *SProxyReader) SetCallback(callback func()) {
	self.callback = callback
}

func (self *SProxyReader) Percent() float64 {
	return float64(self.cur) / float64(self.size) * 100.0
}

func (self *SProxyReader) AvgRate() float64 {
	return float64(self.cur) / float64(1024) / float64(1024) / float64(time.Now().Sub(self.start).Seconds())
}

func (self *SProxyReader) Rate() float64 {
	return float64(self.cur-self.lastCur) / float64(1024) / float64(1024) / float64(self.refreshRate.Seconds())
}

func (self *SProxyReader) Read(p []byte) (n int, err error) {
	defer func() {
		if err != nil {
			self.finish <- true
			close(self.finish)
		}
	}()

	if self.start.IsZero() {
		self.start = time.Now()
		self.ticker = time.NewTicker(self.refreshRate)
		go self.refresh(self.finish)
	}
	n, err = self.reader.Read(p)
	if err != nil {
		return n, err
	}
	if self.rateLimit != nil {
		err = self.rateLimit.WaitN(self.ctx, n)
		if err != nil {
			return n, err
		}
	}
	self.cur += int64(n)
	return n, nil
}

func (self *SProxyReader) refresh(finishChan chan bool) {
	defer self.ticker.Stop()

	for {
		select {
		case <-self.ticker.C:
			if self.callback != nil {
				self.callback()
			}
			self.lastCur = self.cur
		case finished := <-finishChan:
			if finished {
				return
			}
		}
	}
}
