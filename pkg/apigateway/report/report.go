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

package report

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	idapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type sReport struct {
	GenerateName           string
	UUID                   string
	Version                string
	OsDist                 string
	OsVersion              string
	QemuVersion            string
	CpuArchitecture        string
	Brands                 string
	HostCnt                int64
	KvmHostCnt             int64
	VmwareHostCnt          int64
	HostCpuCnt             int64
	HostMemSizeMb          int64
	BaremetalCnt           int64
	BaremetalCpuCnt        int64
	BaremetalMemSizeMb     int64
	BaremetalStorageSizeGb int64
	ServerCnt              int64
	ServerCpuCnt           int64
	ServerMemSizeMb        int64
	DiskCnt                int64
	DiskSizeMb             int64
	BucketCnt              int64
	RdsCnt                 int64
	MongoDBCnt             int64
	KafkaCnt               int64
	UserCnt                int64
	ProjectCnt             int64
	DomainCnt              int64
}

func Report(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	getFunc := func() (*sReport, error) {
		ret := &sReport{}
		ret.Version = version.GetShortString()
		s := auth.GetAdminSession(ctx, options.Options.Region)
		system := jsonutils.Marshal(map[string]string{"scope": "system"})
		user, err := identity.UsersV3.Get(s, idapi.SystemAdminUser, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "Get user %s id", idapi.SystemAdminUser)
		}
		ret.UUID, err = user.GetString("id")
		if err != nil {
			return nil, errors.Wrapf(err, "get user %s id", idapi.SystemAdminUser)
		}
		ret.GenerateName = ret.UUID
		resp, err := compute.Hosts.List(s, jsonutils.Marshal(map[string]interface{}{
			"scope":      "system",
			"hypervisor": api.HYPERVISOR_KVM,
			"limit":      1,
		}))
		if err != nil {
			return nil, errors.Wrapf(err, "Hosts.List")
		}
		ret.KvmHostCnt = int64(resp.Total)

		resp, err = compute.Hosts.List(s, jsonutils.Marshal(map[string]interface{}{
			"scope":      "system",
			"hypervisor": api.HYPERVISOR_ESXI,
			"limit":      1,
		}))
		if err != nil {
			return nil, errors.Wrapf(err, "Hosts.List")
		}
		ret.VmwareHostCnt = int64(resp.Total)

		osDists, osVersions, qemuVersions, archs := []string{}, []string{}, []string{}, []string{}
		hosts := []api.HostDetails{}
		jsonutils.Update(&hosts, resp.Data)
		appendInfo := func(arr []string, info string) []string {
			if len(info) == 0 {
				return arr
			}
			if !utils.IsInStringArray(info, arr) {
				return append(arr, info)
			}
			return arr
		}
		for _, host := range hosts {
			archs = appendInfo(archs, host.CpuArchitecture)
			sn := struct {
				OsDistribution string
				OsVersion      string
				QemuVersion    string
			}{}
			if host.SysInfo != nil {
				host.SysInfo.Unmarshal(&sn)
				osDists = appendInfo(osDists, sn.OsDistribution)
				osVersions = appendInfo(osVersions, sn.OsVersion)
				qemuVersions = appendInfo(qemuVersions, sn.QemuVersion)
			}
		}
		ret.OsDist = strings.Join(osDists, ",")
		ret.OsVersion = strings.Join(osVersions, ",")
		ret.QemuVersion = strings.Join(qemuVersions, ",")
		ret.CpuArchitecture = strings.Join(archs, ",")
		resp, err = compute.Capabilities.List(s, system)
		if err != nil {
			return nil, errors.Wrapf(err, "Capabilities.List")
		}
		if len(resp.Data) > 0 {
			brands := struct {
				Brands         []string
				DisabledBrands []string
			}{}
			resp.Data[0].Unmarshal(&brands)
			ret.Brands = strings.Join(append(brands.Brands, brands.DisabledBrands...), ",")
		}
		usage, err := compute.Usages.GetGeneralUsage(s, system)
		if err != nil {
			return nil, errors.Wrapf(err, "GetGeneralUsage")
		}
		ret.HostCnt, _ = usage.Int("hosts")
		ret.HostCpuCnt, _ = usage.Int("hosts.cpu.total")
		ret.HostMemSizeMb, _ = usage.Int("hosts.memory.total")
		ret.ServerCnt, _ = usage.Int("all.servers")
		ret.ServerCpuCnt, _ = usage.Int("all.servers.cpu")
		ret.ServerMemSizeMb, _ = usage.Int("all.servers.memory")
		ret.DiskCnt, _ = usage.Int("all.disks.count")
		ret.DiskSizeMb, _ = usage.Int("all.disks")
		ret.BucketCnt, _ = usage.Int("all.buckets")
		ret.RdsCnt, _ = usage.Int("all.rds")
		ret.MongoDBCnt, _ = usage.Int("all.mongodb")
		ret.KafkaCnt, _ = usage.Int("all.kafka")
		ret.BaremetalCnt, _ = usage.Int("baremetals")
		ret.BaremetalCpuCnt, _ = usage.Int("baremetals.cpu")
		ret.BaremetalMemSizeMb, _ = usage.Int("baremetals.memory")
		ret.BaremetalStorageSizeGb, _ = usage.Int("baremetals.storage_gb")
		usage, err = identity.IdentityUsages.GetUsage(s, system)
		if err != nil {
			return nil, errors.Wrapf(err, "IdentityUsages.GetUsage")
		}
		ret.UserCnt, _ = usage.Int("users")
		ret.DomainCnt, _ = usage.Int("domains")
		ret.ProjectCnt, _ = usage.Int("projects")
		return ret, nil
	}
	rp, err := func() (*sReport, error) {
		var err error
		var report *sReport
		for i := 0; i < 3; i++ {
			report, err = getFunc()
			if err == nil {
				return report, nil
			}
			time.Sleep(time.Minute * time.Duration(i+1) * 2)
			continue
		}
		return report, err
	}()
	if err != nil {
		log.Errorf("get report error: %v", err)
		return
	}

	url := "https://cloud.yunion.cn/api/v2/opensource-reporting"
	client := httputils.GetDefaultClient()
	_, _, err = httputils.JSONRequest(client, context.Background(), httputils.POST, url, nil, jsonutils.Marshal(rp), false)
	if err != nil {
		log.Errorf("report data error: %v", err)
	}
}
