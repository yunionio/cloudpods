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
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type sInfluxdbEndpointListener struct {
	done map[string]bool
}

func (listener *sInfluxdbEndpointListener) OnServiceCatalogChange(catalog mcclient.IServiceCatalog) {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	urls, err := tsdb.GetDefaultServiceSourceURLs(s, options.Options.SessionEndpointType)
	if err != nil {
		log.Debugf("sInfluxdbEndpointListener: no influxdb endpoints found, retry later...")
		return
	}
	for _, url := range urls {
		if done, ok := listener.done[url]; ok && done {
			continue
		}
		err = setInfluxdbRetentionPolicyForUrl(url)
		if err != nil {
			log.Errorf("set retention policy for url %q fail %s", url, err)
		} else {
			listener.done[url] = true
		}
	}
}

func setInfluxdbRetentionPolicy() {
	listener := &sInfluxdbEndpointListener{
		done: make(map[string]bool),
	}
	auth.RegisterCatalogListener(listener)
}

func setInfluxdbRetentionPolicyForUrl(url string) error {
	db := influxdb.NewInfluxdb(url)
	err := db.SetDatabase("telegraf")
	if err != nil {
		return errors.Wrap(err, "set database telegraf")
	}
	rp := influxdb.SRetentionPolicy{
		Name:     "30day_only",
		Duration: fmt.Sprintf("%dd", options.Options.MetricsRetentionDays),
		ReplicaN: 1,
		Default:  true,
	}
	err = db.SetRetentionPolicy(rp)
	if err != nil {
		return errors.Wrap(err, "set retention policy")
	}
	return nil
}
