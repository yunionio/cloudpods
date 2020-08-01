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
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func setInfluxdbRetentionPolicy() error {
	setF := func() error {
		urls, err := auth.GetServiceURLs("influxdb", options.Options.Region, "", "")
		if err != nil {
			auth.ReAuth()
		}
		if err != nil {
			return errors.Wrap(err, "get influxdb service urls")
		}
		log.Infof("get influxdb service urls: %v", urls)
		for _, url := range urls {
			err = setInfluxdbRetentionPolicyForUrl(url)
			if err != nil {
				return errors.Wrapf(err, "set retention policy for url %q", url)
			}
		}
		return nil
	}

	go func() {
		for {
			err := setF()
			if err == nil {
				log.Infof("setInfluxdbRetentionPolicy completed")
				return
			}
			retryInternal := 1 * time.Minute
			log.Errorf("setInfluxdbRetentionPolicy error: %v, retry after %s", err, retryInternal)
			time.Sleep(retryInternal)
		}
	}()

	return nil
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
