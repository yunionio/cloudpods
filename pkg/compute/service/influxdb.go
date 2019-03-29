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

	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func setInfluxdbRetentionPolicy() error {
	urls, err := auth.GetServiceURLs("influxdb", options.Options.Region, "", "internal")
	if err != nil {
		return err
	}
	for _, url := range urls {
		err = setInfluxdbRetentionPolicyForUrl(url)
		if err != nil {
			return err
		}
	}
	return nil
}

func setInfluxdbRetentionPolicyForUrl(url string) error {
	db := influxdb.NewInfluxdb(url)
	err := db.SetDatabase("telegraf")
	if err != nil {
		return err
	}
	rp := influxdb.SRetentionPolicy{
		Name:     "30day_only",
		Duration: fmt.Sprintf("%dd", options.Options.MetricsRetentionDays),
		ReplicaN: 1,
		Default:  true,
	}
	err = db.SetRetentionPolicy(rp)
	if err != nil {
		return err
	}
	return nil
}
