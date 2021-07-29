package vmwaremon

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SEsxiCloudReport) CollectRegionMetric(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	var err error
	switch self.Operator {
	case string(common.SERVER):
		err = self.collectRegionMetricOfServer(servers)
	case string(common.HOST):
		err = self.collectRegionMetricOfHost(servers)
	}
	return err
}

func (self *SEsxiCloudReport) getMonType() string {
	switch self.Operator {
	case string(common.SERVER):
		return common.TYPE_VIRTUALMACHINE
	case string(common.HOST):
		return common.TYPE_HOSTSYSTEM
	}
	return ""
}

func (self *SEsxiCloudReport) collectRegionMetricOfServerBatch(hostExtId string, servers []jsonutils.JSONObject) error {
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}

	client, err := self.newEsxiClient()
	if err != nil {
		return err
	}
	metrics := make([]string, 0)
	for metric, _ := range esxiMetricSpecs {
		metrics = append(metrics, metric)
	}
	now := time.Now()
	perfEntityMetrics, err := client.GetMonitorDataList(hostExtId, servers, self.getMonType(), metrics, since,
		until)
	if err != nil {
		return errors.Wrap(err, "SEsxiCloudReport GetMonitorDataList error")
	}

	log.Infof("get %s metriclist cost: %f s", self.getMonType(), time.Now().Sub(now).Seconds())

	writeGroup, _ := errgroup.WithContext(context.Background())
	for i, _ := range servers {
		tmpSer := servers[i]
		writeGroup.Go(func() error {
			extId, _ := tmpSer.GetString("external_id")
			dataList := make([]influxdb.SMetricData, 0)
			if entityMetric, ok := perfEntityMetrics[extId]; ok {
				if self.Operator == string(common.SERVER) {
					metric, err := common.FillVMCapacity(tmpSer.(*jsonutils.JSONDict))
					if err != nil {
						return err
					}
					dataList = append(dataList, metric)
				}
				serverMetric := self.collectMetricFromThisServer_(tmpSer, self.getMonType(), entityMetric)
				dataList = append(dataList, serverMetric...)
				writStartTime := time.Now()
				common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
				log.Errorf("influxdb write cost:%f s", time.Now().Sub(writStartTime).Seconds())
			}
			return nil
		})
	}
	err = writeGroup.Wait()
	log.Infof("collect %s num:%d", self.getMonType(), len(servers))
	return nil
}

func (self *SEsxiCloudReport) collectRegionMetricOfServer(servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}

	client, err := self.newEsxiClient()
	if err != nil {
		return err
	}
	for _, server := range servers {
		perfEntityMetrics, metricIdNamedTable, err := client.GetMonitorData(server, common.TYPE_VIRTUALMACHINE,
			esxiMetricSpecsSync, since, until)
		if err != nil {
			log.Errorln(err)
			continue
		}
		for _, perfEntityMetric := range perfEntityMetrics {
			perfMetricSeries := perfEntityMetric.Value
			perfSampleInfos := perfEntityMetric.SampleInfo
			for _, perfMetricSerie := range perfMetricSeries {
				if perfMetricIntSerie, ok := perfMetricSerie.(*types.PerfMetricIntSeries); ok {
					metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
					if err != nil {
						return err
					}
					dataList = append(dataList, metric)
					serverMetric := self.collectMetricFromThisServer(server, common.TYPE_VIRTUALMACHINE, perfMetricIntSerie,
						perfSampleInfos, metricIdNamedTable)
					dataList = append(dataList, serverMetric...)
				}
			}
		}
		err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
		if err != nil {
			log.Errorln(err)
		}
	}
	return nil
}

func (self *SEsxiCloudReport) collectRegionMetricOfHost(hosts []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	client, err := self.newEsxiClient()
	if err != nil {
		return err
	}
	for _, host := range hosts {
		perfEntityMetrics, metricIdNamedTable, err := client.GetMonitorData(host, common.TYPE_HOSTSYSTEM, esxiMetricSpecsSync,
			since, until)
		if err != nil {
			continue
		}
		for _, perfEntityMetric := range perfEntityMetrics {
			perfMetricSeries := perfEntityMetric.Value
			perfSampleInfos := perfEntityMetric.SampleInfo
			for _, perfMetricSerie := range perfMetricSeries {
				if perfMetricIntSerie, ok := perfMetricSerie.(*types.PerfMetricIntSeries); ok {
					serverMetric := self.collectMetricFromThisServer(host, common.TYPE_HOSTSYSTEM, perfMetricIntSerie,
						perfSampleInfos, metricIdNamedTable)
					dataList = append(dataList, serverMetric...)
				}
			}
		}
		err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
		if err != nil {
			log.Errorln(err)
		}
	}
	return nil
}

func (self *SEsxiCloudReport) collectMetricFromThisServer_(server jsonutils.JSONObject, monType string,
	entityMetric performance.EntityMetric) []influxdb.SMetricData {
	datas := make([]influxdb.SMetricData, 0)
	for _, metricSeries := range entityMetric.Value {
		if _, ok := esxiMetricSpecs[metricSeries.Name]; !ok {
			continue
		}
		for i, value := range metricSeries.Value {
			metric := influxdb.SMetricData{}
			if monType == common.TYPE_HOSTSYSTEM {
				metric, _ = common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.HostTags, make([]string, 0))
			} else {
				metric, _ = common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
			}

			if len(entityMetric.SampleInfo) > 0 {
				metric.Timestamp = entityMetric.SampleInfo[i].Timestamp
			}
			influxDbSpecs := esxiMetricSpecs[metricSeries.Name]
			metric.Name = common.GetMeasurement(monType, influxDbSpecs[2])
			var pairsKey string
			if strings.Contains(influxDbSpecs[2], ",") {
				pairsKey = common.SubstringBetween(influxDbSpecs[2], ".", ",")
			} else {
				pairsKey = common.SubstringAfter(influxDbSpecs[2], ".")
			}
			tag := common.SubstringAfter(influxDbSpecs[2], ",")
			if tag != "" && strings.Contains(influxDbSpecs[2], "=") {
				metric.Tags = append(metric.Tags, influxdb.SKeyValue{
					Key:   common.SubstringBefore(tag, "="),
					Value: common.SubstringAfter(tag, "="),
				})
			}
			value = formateValue(value, influxDbSpecs)
			metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
				Key:   pairsKey,
				Value: strconv.FormatInt(value, 10),
			})
			if monType == common.TYPE_HOSTSYSTEM {
				self.AddMetricTag(&metric, common.OtherHostTag)
			} else {
				self.AddMetricTag(&metric, common.OtherVmTags)
			}
			datas = append(datas, metric)
		}
	}
	return datas
}

func (self *SEsxiCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject, monType string,
	perfMetricIntSerie *types.PerfMetricIntSeries, perfSampleInfos []types.PerfSampleInfo,
	metricIdNamedTable map[int32]string) []influxdb.SMetricData {
	datas := make([]influxdb.SMetricData, 0)
	for i, value := range perfMetricIntSerie.Value {
		metric := influxdb.SMetricData{}
		if monType == common.TYPE_HOSTSYSTEM {
			metric, _ = common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.HostTags, make([]string, 0))
		} else {
			metric, _ = common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
		}
		counterId := perfMetricIntSerie.Id.CounterId
		instance := perfMetricIntSerie.Id.Instance
		if instance != "" {
			metric.Tags = append(metric.Tags, influxdb.SKeyValue{
				Key:   "perf_instance",
				Value: instance,
			})
		}
		if len(perfSampleInfos) > 0 {
			metric.Timestamp = perfSampleInfos[i].Timestamp
		}

		influxDbSpecs := esxiMetricSpecsSync[metricIdNamedTable[counterId]]
		metric.Name = common.GetMeasurement(monType, influxDbSpecs[2])
		var pairsKey string
		if strings.Contains(influxDbSpecs[2], ",") {
			pairsKey = common.SubstringBetween(influxDbSpecs[2], ".", ",")
		} else {
			pairsKey = common.SubstringAfter(influxDbSpecs[2], ".")
		}
		tag := common.SubstringAfter(influxDbSpecs[2], ",")
		if tag != "" && strings.Contains(influxDbSpecs[2], "=") {
			metric.Tags = append(metric.Tags, influxdb.SKeyValue{
				Key:   common.SubstringBefore(tag, "="),
				Value: common.SubstringAfter(tag, "="),
			})
		}
		value = formateValue(value, influxDbSpecs)
		metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
			Key:   pairsKey,
			Value: strconv.FormatInt(value, 10),
		})
		if monType == common.TYPE_HOSTSYSTEM {
			self.AddMetricTag(&metric, common.OtherHostTag)
		} else {
			self.AddMetricTag(&metric, common.OtherVmTags)
		}
		datas = append(datas, metric)
	}
	return datas
}

func formateValue(value int64, influxDbSpecs []string) int64 {
	if influxDbSpecs[1] == UNIT_KBPS && strings.Contains(influxDbSpecs[2], "bps") {
		value = value * 1000
	}
	if influxDbSpecs[1] == UNIT_KBPS && strings.Contains(influxDbSpecs[2], "bytes") {
		value = value * 1000
	}
	if influxDbSpecs[1] == UNIT_PERCENT && strings.Contains(influxDbSpecs[2], "usage_active") {
		value = value / 100
	}
	if influxDbSpecs[1] == UNIT_PERCENT && strings.Contains(influxDbSpecs[2], "used_percent") {
		value = value / 100
	}
	return value
}

func (self *SEsxiCloudReport) newEsxiClient() (*esxi.SESXiClient, error) {
	parts, err := url.Parse(self.SProvider.AccessUrl)
	if err != nil {
		return nil, err
	}
	host, port, err := parseHostPort(parts.Host, 443)
	if err != nil {
		return nil, err
	}

	secretDe, _ := utils.DescryptAESBase64(self.SProvider.Id, self.SProvider.Secret)
	esxiCfg := esxi.NewESXiClientConfig(host, port, self.SProvider.Account, secretDe)
	proCfg := cloudprovider.ProviderConfig{
		Id:      self.SProvider.Id,
		Name:    self.SProvider.Name,
		Account: self.SProvider.Account,
		Secret:  secretDe,
		URL:     self.SProvider.AccessUrl,
		Vendor:  self.SProvider.Provider,
	}
	esxiCfg.CloudproviderConfig(proCfg)
	client, err := esxi.NewESXiClient(esxiCfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func parseHostPort(host string, defPort int) (string, int, error) {
	colonPos := strings.IndexByte(host, ':')
	if colonPos > 0 {
		h := host[:colonPos]
		p, err := strconv.Atoi(host[colonPos+1:])
		if err != nil {
			log.Errorf("Invalid host %s", host)
			return "", 0, err
		}
		if p == 0 {
			p = defPort
		}
		return h, p, nil
	} else {
		return host, defPort, nil
	}
}
