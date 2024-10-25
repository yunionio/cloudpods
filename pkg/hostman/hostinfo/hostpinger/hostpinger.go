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

package hostpinger

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/mem"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SHostPingTask struct {
	interval int // second
	running  bool
	host     hostutils.IHost

	// masterHostStorages for shared storages
	masterHostStorages []string
	lastStatAt         time.Time
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

func NewHostPingTask(interval int, host hostutils.IHost) *SHostPingTask {
	if interval <= 0 {
		return nil
	}
	return &SHostPingTask{
		interval: interval,
		running:  true,
		host:     host,
	}
}

func (p *SHostPingTask) Start() {
	log.Infof("Start host pinger ...")
	var (
		div    = 1
		hostId = p.host.GetHostId()
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

func (p *SHostPingTask) payload() api.SHostPingInput {
	data := api.SHostPingInput{}

	now := time.Now()
	if !p.lastStatAt.IsZero() && now.Before(p.lastStatAt.Add(time.Duration(options.HostOptions.SyncStorageInfoDurationSecond)*time.Second)) {
		return data
	}

	p.lastStatAt = now
	data = storageman.GatherHostStorageStats(p.masterHostStorages)
	data.WithData = true
	info, err := mem.VirtualMemory()
	if err != nil {
		return data
	}
	memTotal := int(info.Total / 1024 / 1024)
	memFree := int(info.Available / 1024 / 1024)
	memUsed := memTotal - memFree
	data.MemoryUsedMb = memUsed
	data.QgaRunningGuestIds = guestman.GetGuestManager().GetQgaRunningGuests()
	return data
}

func (p *SHostPingTask) ping(div int, hostId string) error {
	log.Debugf("ping region at %d...", div)
	res, err := modules.Hosts.PerformAction(hostutils.GetComputeSession(context.Background()),
		hostId, "ping", jsonutils.Marshal(p.payload()))
	if err != nil {
		if errors.Cause(err) == httperrors.ErrResourceNotFound {
			log.Errorf("host seemd removed from region ...")
			return nil
		} else {
			return errors.Wrap(err, "ping")
		}
	} else {
		// name, err := res.GetString("name")
		// if err != nil {
		// 	Instance().setHostname(name)
		// }

		if res.Contains("master_host_storages") {
			storages := make([]string, 0)
			res.Unmarshal(&storages, "master_host_storages")
			p.masterHostStorages = storages
		}

		catalog, err := res.Get("catalog")
		if err == nil {
			cl := make(mcclient.KeystoneServiceCatalogV3, 0)
			err = catalog.Unmarshal(&cl)
			if err != nil {
				log.Errorln(err)
				return nil
			}

			p.host.OnCatalogChanged(cl)
		} else {
			log.Errorf("get catalog from res %s: %v", res.String(), err)
		}
	}
	return nil
}

func (p *SHostPingTask) Stop() {
	if p.running {
		p.running = false
	}
}
