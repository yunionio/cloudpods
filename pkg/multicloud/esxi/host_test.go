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

package esxi

import (
	"fmt"
	"testing"

	"github.com/vmware/govmomi/vim25/mo"

	"yunion.io/x/log"
)

func TestFormatName(t *testing.T) {
	cases := []struct {
		In   string
		Want string
	}{
		{
			In:   "esxi-172.16.23.1",
			Want: "esxi-172-16-23-1",
		},
		{
			In:   "esxi6.yunion.cn",
			Want: "esxi6",
		},
	}
	for _, c := range cases {
		got := formatName(c.In)
		if got != c.Want {
			t.Errorf("got: %s want %s", got, c.Want)
		}
	}
}

func TestSESXiClient_FindHostByIp(t *testing.T) {
	host.getNicInfo()
	portgroups := make([]mo.DistributedVirtualPortgroup, 0)
	err := host.manager.references2Objects(host.getHostSystem().Network, DVPORTGROUP_PROPS, &portgroups)
	if err != nil {
		log.Errorf(err.Error())
		return
	}

	for _, p := range portgroups {
		fmt.Printf("%v\n", p)
	}
}

func TestSESXiClient_GetDatacenters(t *testing.T) {
	_, err := dc.GetNetworks()
	if err != nil {
		return
	}
	odc := dc.getDatacenter()
	fmt.Printf("%s: %s", odc.VmFolder.Type, odc.VmFolder.Value)
	portgroups := make([]mo.DistributedVirtualPortgroup, 0)
	err = dc.manager.references2Objects(dc.getDatacenter().Network, DVPORTGROUP_PROPS, &portgroups)
	if err != nil {
		log.Errorf(err.Error())
		return
	}

	for _, p := range portgroups {
		fmt.Printf("%v\n", p)
	}
}

func TestSDistributedVirtualPortgroup_AddHostToDVS(t *testing.T) {
	nets, err := dc.GetNetworks()
	if err != nil {
		return
	}
	for _, net := range nets {
		if !net.ContainHost(host) {
			dvgp, ok := net.(*SDistributedVirtualPortgroup)
			if ok {
				err := dvgp.AddHostToDVS(host)
				if err != nil {
					log.Errorf(err.Error())
					return
				}
			}
		}
	}
}

var (
	ip, account, passwd string
	sc                  *SESXiClient
	dc                  *SDatacenter
	host                *SHost
)

func TestMain(m *testing.M) {
	var err error
	ip := "192.168.222.202"
	account := "administrator@vsphere.local"
	passwd := "123@VMware"
	sc, err = NewESXiClient("", "", ip, 0, account, passwd)
	if err != nil {
		log.Errorf("fail to init ESXiClient")
		return
	}
	dcs, err := sc.GetDatacenters()
	if err != nil {
		log.Errorf(err.Error())
		return
	}
	// avoid to occupy memory
	dc = dcs[:1][0]
	hostIp := "192.168.222.201"
	host, err = sc.FindHostByIp(hostIp)
	if err != nil {
		log.Errorf(err.Error())
		return
	}
	m.Run()
}
