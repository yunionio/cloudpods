package service

import (
	"context"
	"os"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	common_app "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

var (
	cloudproviderChanMap    = make(map[string]chan struct{})
	cutomizeOperatorChanmap = make(map[string]chan struct{})
)

func StartService() {
	opts := &options.Options
	common_options.ParseOptions(opts, os.Args, "cloudmon.conf", "cloudmon")

	commonOpts := &opts.CommonOptions
	common_app.InitAuth(commonOpts, func() {
		log.Infof("Auth complete")
	})
	ctx := context.Background()
	duration := time.Duration(opts.CloudproviderSyncInterval) * time.Minute
	log.Errorf("CloudproviderSyncInterval: %d", opts.CloudproviderSyncInterval)
	nextSync := time.Now()
	for i := range common.CustomizeMonTypeList {
		monType := common.CustomizeMonTypeList[i]
		cutomizeOperatorChanmap[monType] = make(chan struct{})
		cloudReportFactory, err := common.GetCloudReportFactory(monType)
		if err != nil {
			log.Errorf("CustomizeMonType: %s GetCloudReportFactory err: %v", monType, err)
			continue
		}
		if routineF, ok := cloudReportFactory.(common.IRoutineFactory); ok {
			routineF.MyRoutineFunc()(ctx, cloudReportFactory, nil, cutomizeOperatorChanmap[monType],
				cloudReportFactory.MyRoutineInteval(*opts), common.CustomizeRunFunc)
			continue
		}
		common.MakePullMetricRoutineWithDur(ctx, cloudReportFactory, nil, cutomizeOperatorChanmap[monType],
			cloudReportFactory.MyRoutineInteval(*opts), common.CustomizeRunFunc)
	}

	for {
		if nextSync.Before(time.Now()) {
			cloudproviderList, err := getCloudproviderList(ctx)
			if err != nil {
				log.Errorf("err: %v", err)
				return
			}
			cloudproviderList = syncCloudproviderChanMap(cloudproviderList)
			log.Errorf("cloudproviderList: %d", len(cloudproviderList))
			for i := 0; i < len(cloudproviderList); i++ {
				provider := cloudproviderList[i]
				status, err := provider.GetString("status")
				if err != nil {
					log.Errorf("provider: %v get status error: %v", provider, err)
					continue
				}
				if status == "connected" {
					id, _ := provider.GetString("id")
					err := common.ReportConnectCloudproviderMetric(ctx, provider, cloudproviderChanMap[id])
					if err != nil {
						log.Errorf("ReportConnectCloudproviderMetric err: %v", err)
					}
				}
			}
			nextSync = time.Now().Add(duration)
		}
		time.Sleep(time.Minute)
	}

}

func getCloudproviderList(ctx context.Context) ([]jsonutils.JSONObject, error) {
	session := auth.GetAdminSession(ctx, "")
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("10"), common.KEY_LIMIT)
	query.Add(jsonutils.NewString("true"), common.KEY_ADMIN)
	//query.Add(jsonutils.NewString("true"), KEY_USABLE)
	if len(common.SupportMetricBrands) == 0 {
		return nil, errors.Errorf("SupportMetricBrands is empty")
	}
	cloudProviderList := make([]jsonutils.JSONObject, 0)
	for _, val := range common.SupportMetricBrands {
		query.Add(jsonutils.NewString(val), "provider")
		tmpList, err := common.ListAllResources(&modules.Cloudproviders, session, query)
		if err != nil {
			return nil, errors.Wrap(err, "list Cloudproviders err")
		}
		cloudProviderList = append(cloudProviderList, tmpList...)
	}
	return cloudProviderList, nil
}

func syncCloudproviderChanMap(cloudproviderList []jsonutils.JSONObject) []jsonutils.JSONObject {
	newCloudprovider := make([]jsonutils.JSONObject, 0)
	providerIds := make([]string, 0)
	for i, _ := range cloudproviderList {
		id, _ := cloudproviderList[i].GetString("id")
		providerIds = append(providerIds, id)
		if _, ok := cloudproviderChanMap[id]; ok {
			continue
		}
		newCloudprovider = append(newCloudprovider, cloudproviderList[i])
		cloudproviderChanMap[id] = make(chan struct{})
	}
	for id, channel := range cloudproviderChanMap {
		if utils.IsInStringArray(id, providerIds) {
			continue
		}
		delete(cloudproviderChanMap, id)
		close(channel)
	}
	return newCloudprovider
}
