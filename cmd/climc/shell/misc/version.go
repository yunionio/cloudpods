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

package misc

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func init() {
	type VersionOptions struct {
		SERVICE string `help:"Service type"`
	}
	R(&VersionOptions{}, "version-show", "query backend service for its version", func(s *mcclient.ClientSession, args *VersionOptions) error {
		body, err := modulebase.GetVersion(s, args.SERVICE)
		if err != nil {
			return err
		}
		fmt.Println(body)
		return nil
	})

	type StatsOptions struct {
		SERVICE string `help:"Service type"`
	}
	R(&StatsOptions{}, "api-stats-show", "query backend service for its stats", func(s *mcclient.ClientSession, args *StatsOptions) error {
		body, err := modulebase.GetStats(s, "stats", args.SERVICE)
		if err != nil {
			return err
		}
		printObject(body)
		return nil
	})
	R(&StatsOptions{}, "db-stats-show", "query backend service for its db stats", func(s *mcclient.ClientSession, args *StatsOptions) error {
		body, err := modulebase.GetStats(s, "db_stats", args.SERVICE)
		if err != nil {
			return err
		}
		stats, _ := body.Get("db_stats")
		printObject(stats)
		return nil
	})
	R(&StatsOptions{}, "worker-stats-show", "query backend service for its worker stats", func(s *mcclient.ClientSession, args *StatsOptions) error {
		body, err := modulebase.GetStats(s, "worker_stats", args.SERVICE)
		if err != nil {
			return err
		}
		data, _ := body.GetArray("workers")
		printList(&printutils.ListResult{Data: data}, nil)
		return nil
	})

	type VersionListOptions struct {
	}
	R(&VersionListOptions{}, "version-list", "query all backend service version", func(s *mcclient.ClientSession, args *VersionListOptions) error {
		services := []jsonutils.JSONObject{}
		params := map[string]interface{}{}
		for {
			params["offset"] = len(services)
			resp, err := identity.ServicesV3.List(s, jsonutils.Marshal(params))
			if err != nil {
				return err
			}
			services = append(services, resp.Data...)
			if len(services) >= resp.Total {
				break
			}
		}
		vers := map[string]string{}
		for _, service := range services {
			serviceType, _ := service.GetString("type")
			if utils.IsInStringArray(serviceType, []string{
				apis.SERVICE_TYPE_OFFLINE_CLOUDMETA,
				apis.SERVICE_TYPE_CLOUDMETA,
				apis.SERVICE_TYPE_INFLUXDB,
				apis.SERVICE_TYPE_VICTORIA_METRICS,
				apis.SERVICE_TYPE_ETCD,
				"torrent-tracker",
			}) {
				continue
			}
			ver, err := modules.GetVersion(s, serviceType)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound || errors.Cause(err) == errors.ErrConnectRefused {
					continue
				}
				vers[serviceType] = err.Error()
			} else {
				vers[serviceType] = ver
			}
		}
		printObject(jsonutils.Marshal(vers))
		return nil
	})
}
