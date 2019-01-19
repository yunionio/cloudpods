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
	)
	for {
		time.Sleep(time.Duration(p.interval/div) * time.Second)
		if !p.running {
			return
		}
		res, err := modules.Hosts.PerformAction(hostutils.GetComputeSession(context.Background()),
			hostId, "ping", nil)
		if err != nil {
			div = 3
		} else {
			name, err := res.GetString("name")
			if err != nil {
				Instance().setHostname(name)
			}
			catalog, err := res.Get("catalog")
			if err != nil {
				cl := make(mcclient.KeystoneServiceCatalogV3, 0)
				err = catalog.Unmarshal(&cl)
				if err != nil {
					log.Errorln(err)
					continue
				}

				Instance().OnCatalogChanged(cl)
			}
		}
	}
}

func (p *SHostPingTask) Stop() {
	if p.running {
		p.running = false
	}
}
