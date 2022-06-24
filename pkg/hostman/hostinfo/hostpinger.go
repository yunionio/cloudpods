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

package hostinfo

import (
	"context"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SHostPingTask struct {
	interval int // second
	running  bool
}

type SEndpoint struct {
	Id        string `json:"id"`
	Interface string `json:"interface"`
	Region    string `json:"region"`
	Region_id string `json:"region_id"`
	Url       string `json:"url"`
	Name      string `json:"name"`
}

type SCatalog struct {
	Id        string      `json:"id"`
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Endpoints []SEndpoint `json:"endpoint"`
}

func NewCatalog() *SCatalog {
	return &SCatalog{
		Endpoints: make([]SEndpoint, 0),
	}
}

func NewHostPingTask(interval int) *SHostPingTask {
	if interval <= 0 {
		return nil
	}
	return &SHostPingTask{interval, true}
}

func (p *SHostPingTask) Start() {
	var (
		div    = 1
		hostId = Instance().GetHostId()
		err    error
	)
	for {
		if !p.running {
			return
		}
		if err = p.ping(div, hostId); err != nil {
			log.Errorf("host ping failed %s", err)
			div = 3
		} else {
			div = 1
		}

		time.Sleep(time.Duration(p.interval/div) * time.Second)
	}
}

func (p *SHostPingTask) ping(div int, hostId string) error {
	res, err := modules.Hosts.PerformAction(hostutils.GetComputeSession(context.Background()),
		hostId, "ping", nil)
	if err != nil {
		return err
	} else {
		// name, err := res.GetString("name")
		// if err != nil {
		// 	Instance().setHostname(name)
		// }
		catalog, err := res.Get("catalog")
		if err == nil {
			cl := make(mcclient.KeystoneServiceCatalogV3, 0)
			err = catalog.Unmarshal(&cl)
			if err != nil {
				log.Errorln(err)
				return nil
			}

			Instance().OnCatalogChanged(cl)
		} else {
			log.Errorln(err)
		}
	}
	return nil
}

func (p *SHostPingTask) Stop() {
	if p.running {
		p.running = false
	}
}
