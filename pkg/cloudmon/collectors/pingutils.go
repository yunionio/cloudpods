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

package collectors

import (
	"fmt"
	"net"
	"time"

	"github.com/tatsushid/go-fastping"

	"yunion.io/x/pkg/errors"
)

type SPingResult struct {
	addr  string
	rtt   []time.Duration
	count int
}

func NewPingResult(addr string, count int) *SPingResult {
	return &SPingResult{
		addr:  addr,
		rtt:   make([]time.Duration, 0),
		count: count,
	}
}

func (pr *SPingResult) Add(rtt time.Duration) {
	pr.rtt = append(pr.rtt, rtt)
}

func (pr SPingResult) Rtt() (min time.Duration, avg time.Duration, max time.Duration) {
	sum := time.Duration(0)
	max = time.Duration(-1)
	min = time.Duration(-1)
	for _, d := range pr.rtt {
		sum += d
		if max < 0 || max < d {
			max = d
		}
		if min < 0 || min > d {
			min = d
		}
	}
	if len(pr.rtt) > 0 {
		avg = sum / time.Duration(len(pr.rtt))
	}
	return
}

func (pr SPingResult) Loss() int {
	return 100 - len(pr.rtt)*100/pr.count
}

func (pr SPingResult) String() string {
	min, avg, max := pr.Rtt()
	return fmt.Sprintf("%d packets transmitted, %d received, %d%% packet loss, rtt min/avg/max = %d/%d/%d ms", pr.count, len(pr.rtt), pr.Loss(), min/time.Millisecond, avg/time.Millisecond, max/time.Millisecond)
}

func Ping(addrList []string, count int, timeout time.Duration, debug bool) (map[string]*SPingResult, error) {
	p := fastping.NewPinger()
	p.MaxRTT = timeout
	p.Size = 64
	p.Debug = debug
	result := make(map[string]*SPingResult)
	for _, addr := range addrList {
		result[addr] = NewPingResult(addr, count)
		p.AddIP(addr)
	}
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		result[addr.String()].Add(rtt)
	}
	p.OnIdle = func() {
	}
	p.RunLoop()
	defer p.Stop()
	ticker := time.NewTicker(time.Duration(count) * timeout)
	defer ticker.Stop()
	select {
	case <-p.Done():
		if err := p.Err(); err != nil {
			return nil, errors.Wrap(err, "ping error")
		}
		break
	case <-ticker.C:
		break
	}
	return result, nil
}
