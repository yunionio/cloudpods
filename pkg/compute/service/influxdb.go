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
