package service

import (
	"context"
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/httputils"
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
	query := fmt.Sprintf("CREATE DATABASE telegraf; CREATE RETENTION POLICY \"30day_only\" ON \"telegraf\" DURATION %dd REPLICATION 1 DEFAULT", options.Options.MetricsRetentionDays)
	queryDict := jsonutils.NewDict()
	queryDict.Add(jsonutils.NewString(query), "q")
	nurl := fmt.Sprintf("%s/query?%s", url, queryDict.QueryString())
	resp, err := httputils.Request(httputils.GetDefaultClient(), context.Background(), "POST", nurl, nil, nil, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Debugf("setInfluxdbRetentionPolicyForUrl: %s %s", url, string(respBody))
	return nil
}
