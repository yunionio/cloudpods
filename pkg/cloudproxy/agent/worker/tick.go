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

package worker

import (
	"context"
	"math/rand"
	"time"

	"yunion.io/x/log"

	agentssh "yunion.io/x/onecloud/pkg/cloudproxy/agent/ssh"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	cloudproxy_modules "yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
)

func tickDuration(timeout int) time.Duration {
	if timeout > 30 {
		return time.Duration(timeout-5-rand.Intn(10)) * time.Second
	}
	return (time.Duration(timeout) * time.Second / 3) * 2
}

func heartbeatFunc(fwdId string, sessionCache *auth.SessionCache) agentssh.TickFunc {
	return func(ctx context.Context) {
		s := sessionCache.Get(ctx)
		_, err := cloudproxy_modules.Forwards.PerformAction(s, fwdId, "heartbeat", nil)
		if err != nil {
			log.Errorf("forwarder heartbeat: %s: %v", fwdId, err)
		}
	}
}
