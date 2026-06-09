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

package service

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/aiproxy/options"
	"yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
)

func startSlaveNodeRegisterLoop(opts *options.SAiProxyOptions) {
	interval := time.Duration(opts.NodeHeartbeatIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	address, err := models.AdvertiseAddressFromOptions(opts)
	if err != nil {
		log.Fatalf("invalid standby advertise address: %v", err)
	}

	go func() {
		ctx := context.Background()
		register := func() {
			session := auth.GetAdminSessionWithPublic(ctx, opts.Region)
			master, err := session.GetServiceURL(apmodules.AiProxyNodes.ServiceType(), identity.EndpointInterfacePublic, httputils.POST)
			if err != nil {
				log.Errorf("aiproxy standby resolve primary public endpoint failed: %v", err)
				return
			}
			params := jsonutils.Marshal(map[string]interface{}{
				"address": address,
			})
			if _, err := apmodules.AiProxyNodes.PerformClassAction(session, "register", params); err != nil {
				log.Errorf("aiproxy standby register with primary %s failed: %v", master, err)
				return
			}
			log.Debugf("aiproxy standby registered with primary %s as %s", master, address)
		}
		register()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			register()
		}
	}()
}
