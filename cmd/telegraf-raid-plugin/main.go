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

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/baremetal/utils/detect_storages"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid/drivers"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	INTERVAL_SECOND = 300
	TelegrafServer  = "http://localhost:8087/write"
)

// Failed, Offline, Degraded, Rebuilding, Out of Sync (OSY)

func main() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalln(err)
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	nodeName := os.Getenv("NODENAME")
	if len(nodeName) == 0 {
		log.Fatalf("Missing env nodename")
	}

	node, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	var masterAddress string
	if length := len(node.Status.Conditions); length > 0 {
		if node.Status.Conditions[length-1].Type == v1.NodeReady &&
			node.Status.Conditions[length-1].Status == v1.ConditionTrue {
			for _, addr := range node.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					masterAddress = addr.Address
					break
				}
			}
		}
	}

	log.Infof("Start Colloct Raid Info And Send To Telegraf ...")
	c := NewRaidInfoCollector(nodeName, masterAddress, INTERVAL_SECOND)
	c.Start()
}

type RaidInfoCollector struct {
	waitingReportData []string
	LastCollectTime   time.Time
	ReportInterval    int // seconds
	Hostname          string
	HostIp            string
}

func NewRaidInfoCollector(hostname, hostIp string, interval int) *RaidInfoCollector {
	return &RaidInfoCollector{
		waitingReportData: make([]string, 0),
		ReportInterval:    interval,
		Hostname:          hostname,
		HostIp:            hostIp,
	}
}

func (c *RaidInfoCollector) runMain() {
	timeBegin := time.Now()
	elapse := timeBegin.Sub(c.LastCollectTime)
	if elapse < time.Second*time.Duration(c.ReportInterval) {
		return
	} else {
		c.LastCollectTime = timeBegin
	}
	c.runMonitor()
}

func (c *RaidInfoCollector) runMonitor() {
	reportData := c.collectReportData()
	if len(reportData) > 0 {
		c.reportRaidInfoToTelegraf(reportData)
	}
}

func (c *RaidInfoCollector) collectReportData() string {
	if len(c.waitingReportData) > 60 {
		c.waitingReportData = c.waitingReportData[1:]
	}
	return c.CollectReportData()
}

func (c *RaidInfoCollector) CollectReportData() string {
	raidDiskInfo := make([]*baremetal.BaremetalStorage, 0)
	// raidDrivers := []string{}
	for _, drv := range drivers.GetDrivers(drivers.NewExecutor()) {
		if err := drv.ParsePhyDevs(); err != nil {
			log.Warningf("Raid driver %s ParsePhyDevs failed: %s", drv.GetName(), err)
			continue
		}
		raidDiskInfo = append(raidDiskInfo, detect_storages.GetRaidDevices(drv)...)
		// raidDrivers = append(raidDrivers, drv.GetName())
	}
	if len(raidDiskInfo) > 0 {
		ret := c.toTelegrafReportData(raidDiskInfo)
		return ret
	}
	return ""
}

func (c *RaidInfoCollector) Start() {
	for {
		c.runMain()
		time.Sleep(time.Second * 1)
	}
}

const MEASUREMENT = "host_raid"

func (c *RaidInfoCollector) toTelegrafReportData(raidDiskInfo []*baremetal.BaremetalStorage) string {
	tag := fmt.Sprintf("%s=%s,%s=%s", "hostname", c.Hostname, "host_ip", c.HostIp)
	ret := []string{}
	for i := 0; i < len(raidDiskInfo); i++ {
		statArr := []string{}
		raidDiskInfo[i].Status = strings.ToLower(raidDiskInfo[i].Status)
		jStat := jsonutils.Marshal(raidDiskInfo[i])
		jMap, _ := jStat.GetMap()
		for k, v := range jMap {
			statArr = append(statArr, fmt.Sprintf("%s=%s", k, v.String()))
		}
		stat := strings.Join(statArr, ",")
		diskTag := fmt.Sprintf(
			"%s,%s=%s,%s=%d,%s=%d", tag, "driver", raidDiskInfo[i].Driver,
			"adapter", raidDiskInfo[i].Adapter, "slot", raidDiskInfo[i].Slot,
		)
		line := fmt.Sprintf("%s,%s %s", MEASUREMENT, diskTag, stat)
		ret = append(ret, line)
	}
	return strings.Join(ret, "\n")
}

func (c *RaidInfoCollector) reportRaidInfoToTelegraf(data string) {
	body := strings.NewReader(data)
	res, err := httputils.Request(
		httputils.GetDefaultClient(), context.Background(), "POST", TelegrafServer, nil, body, false)
	if err != nil {
		log.Errorf("Upload guest metric failed: %s", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 204 {
		log.Errorf("upload guest metric failed %d", res.StatusCode)
		timestamp := time.Now().UnixNano()
		for _, line := range strings.Split(data, "\n") {
			c.waitingReportData = append(c.waitingReportData,
				fmt.Sprintf("%s %d", line, timestamp))
		}
	} else {
		if len(c.waitingReportData) > 0 {
			oldDatas := strings.Join(c.waitingReportData, "\n")
			body = strings.NewReader(oldDatas)
			res, err = httputils.Request(
				httputils.GetDefaultClient(), context.Background(), "POST", TelegrafServer, nil, body, false)
			if err == nil {
				defer res.Body.Close()
			}
			if res.StatusCode == 204 {
				c.waitingReportData = c.waitingReportData[len(c.waitingReportData):]
			} else {
				log.Errorf("upload guest metric failed code: %d", res.StatusCode)
			}
		}
	}
}
