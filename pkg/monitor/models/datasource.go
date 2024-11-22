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

package models

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/influxdata/promql/v2/pkg/labels"
	"github.com/zexi/influxql-to-metricsql/converter/translator"
	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/datasource"
	merrors "yunion.io/x/onecloud/pkg/monitor/errors"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/validators"
	"yunion.io/x/onecloud/pkg/util/influxdb"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	VICTORIA_METRICS_DB_TAG_KEY          = "db"
	VICTORIA_METRICS_DB_TAG_VAL_TELEGRAF = "telegraf"
)

var (
	DataSourceManager *SDataSourceManager
	compile           = regexp.MustCompile(`\w{8}(-\w{4}){3}-\w{12}`)
)

func init() {
	DataSourceManager = &SDataSourceManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SDataSource{},
			"datasources_tbl",
			"datasource",
			"datasources",
		),
	}
	DataSourceManager.SetVirtualObject(DataSourceManager)
}

type SDataSourceManager struct {
	db.SStandaloneResourceBaseManager
}

type SDataSource struct {
	db.SStandaloneResourceBase

	Type      string            `nullable:"false" list:"user"`
	Url       string            `nullable:"false" list:"user"`
	User      string            `width:"64" charset:"utf8" nullable:"true"`
	Password  string            `width:"64" charset:"utf8" nullable:"true"`
	Database  string            `width:"64" charset:"utf8" nullable:"true"`
	IsDefault tristate.TriState `default:"false" create:"optional"`
	/*
		TimeInterval string
		BasicAuth bool
		BasicAuthUser string
		BasicAuthPassword string
	*/
}

func (m *SDataSourceManager) GetSource(id string) (*SDataSource, error) {
	ret, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return ret.(*SDataSource), nil
}

func (m *SDataSourceManager) GetDatabases() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	dataSource, err := datasource.GetDefaultSource("")
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetDefaultSource")
	}
	db := influxdb.NewInfluxdb(dataSource.Url)
	//db.SetDatabase("telegraf")
	databases, err := db.GetDatabases()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "GetDatabases")
	}
	ret.Add(jsonutils.NewStringArray(databases), "databases")
	return ret, nil
}

func (m *SDataSourceManager) GetMeasurements(query jsonutils.JSONObject,
	measurementFilter, tagFilter string) (jsonutils.JSONObject,
	error) {
	ret := jsonutils.NewDict()
	measurements, err := m.getMeasurementQueryInfluxdb(query, measurementFilter, tagFilter)
	if err != nil {
		return jsonutils.JSONNull, err
	}
	ret.Add(jsonutils.Marshal(&measurements), "measurements")
	return ret, nil
}

func (m *SDataSourceManager) getMeasurementQueryInfluxdb(query jsonutils.JSONObject,
	measurementFilter, tagFilter string) (rtnMeasurements []monitor.InfluxMeasurement, err error) {
	database, _ := query.GetString("database")
	if database == "" {
		return rtnMeasurements, merrors.NewArgIsEmptyErr("database")
	}
	dataSource, err := datasource.GetDefaultSource("")
	if err != nil {
		return rtnMeasurements, errors.Wrap(err, "s.GetDefaultSource")
	}
	db := influxdb.NewInfluxdb(dataSource.Url)
	db.SetDatabase(database)
	var buffer bytes.Buffer
	buffer.WriteString(" SHOW MEASUREMENTS ON ")
	buffer.WriteString(database)
	if len(measurementFilter) != 0 {
		buffer.WriteString(" WITH ")
		buffer.WriteString(measurementFilter)
	}
	if len(tagFilter) != 0 {
		buffer.WriteString(" WHERE ")
		buffer.WriteString(tagFilter)
	}
	dbRtn, err := db.Query(buffer.String())
	if err != nil {
		return rtnMeasurements, errors.Wrap(err, "SHOW MEASUREMENTS")
	}
	if len(dbRtn) > 0 && len(dbRtn[0]) > 0 {
		res := dbRtn[0][0]
		measurements := make([]monitor.InfluxMeasurement, len(res.Values))
		for i := range res.Values {
			tmpDict := jsonutils.NewDict()
			tmpDict.Add(res.Values[i][0], "measurement")
			err = tmpDict.Unmarshal(&measurements[i])
			if err != nil {
				return rtnMeasurements, errors.Wrap(err, "measurement unmarshal error")
			}
		}
		rtnMeasurements = append(rtnMeasurements, measurements...)
	}
	return
}

func (m *SDataSourceManager) GetMeasurementsWithDescriptionInfos(query jsonutils.JSONObject, tagFilter *monitor.MetricQueryTag) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	rtnMeasurements := make([]monitor.InfluxMeasurement, 0)
	measurements, err := MetricMeasurementManager.getMeasurementsFromDB()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "getMeasurementsFromDB")
	}
	filterMeasurements, err := m.filterMeasurementsByTime(measurements, query, tagFilter)
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "filterMeasurementsByTime error")
	}
	filterMeasurements = m.getMetricDescriptions(filterMeasurements)
	if len(filterMeasurements) != 0 {
		rtnMeasurements = append(rtnMeasurements, filterMeasurements...)
	}

	ret.Add(jsonutils.Marshal(&rtnMeasurements), "measurements")
	resTypeMap := make(map[string][]monitor.InfluxMeasurement, 0)
	resTypes := make([]string, 0)
	for _, measurement := range rtnMeasurements {
		if typeMeasurements, ok := resTypeMap[measurement.ResType]; ok {
			resTypeMap[measurement.ResType] = append(typeMeasurements, measurement)
			continue
		}
		resTypes = append(resTypes, measurement.ResType)
		resTypeMap[measurement.ResType] = []monitor.InfluxMeasurement{measurement}
	}
	sort.Slice(resTypes, func(i, j int) bool {
		r1 := resTypes[i]
		r2 := resTypes[j]
		return monitor.ResTypeScoreMap[r1] < monitor.ResTypeScoreMap[r2]
	})
	for _, measures := range resTypeMap {
		sort.Slice(measures, func(i, j int) bool {
			return measures[i].Score < measures[j].Score
		})
	}
	ret.Add(jsonutils.Marshal(&resTypes), "res_types")
	ret.Add(jsonutils.Marshal(&resTypeMap), "res_type_measurements")
	return ret, nil
}

func (m *SDataSourceManager) GetMeasurementsWithOutTimeFilter(query jsonutils.JSONObject,
	measurementFilter, tagFilter string) (jsonutils.JSONObject,
	error) {
	ret := jsonutils.NewDict()
	database, _ := query.GetString("database")
	if database == "" {
		return jsonutils.JSONNull, httperrors.NewInputParameterError("not support database")
	}
	dataSource, err := datasource.GetDefaultSource("")
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetDefaultSource")
	}
	db := influxdb.NewInfluxdb(dataSource.Url)
	db.SetDatabase(database)
	var buffer bytes.Buffer
	buffer.WriteString(" SHOW MEASUREMENTS ON ")
	buffer.WriteString(database)
	if len(measurementFilter) != 0 {
		buffer.WriteString(" WITH ")
		buffer.WriteString(measurementFilter)
	}
	if len(tagFilter) != 0 {
		buffer.WriteString(" WHERE ")
		buffer.WriteString(tagFilter)
	}
	dbRtn, err := db.Query(buffer.String())
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "SHOW MEASUREMENTS")
	}
	if len(dbRtn) > 0 && len(dbRtn[0]) > 0 {
		res := dbRtn[0][0]
		measurements := make([]monitor.InfluxMeasurement, len(res.Values))
		for i := range res.Values {
			tmpDict := jsonutils.NewDict()
			tmpDict.Add(res.Values[i][0], "measurement")
			err := tmpDict.Unmarshal(&measurements[i])
			if err != nil {
				return jsonutils.JSONNull, errors.Wrap(err, "measurement unmarshal error")
			}
		}
		ret.Add(jsonutils.Marshal(&measurements), "measurements")
	}
	return ret, nil
}

func (m *SDataSourceManager) getMetricDescriptions(influxdbMeasurements []monitor.InfluxMeasurement) (
	descMeasurements []monitor.InfluxMeasurement) {
	userCred := auth.AdminCredential()
	listInput := new(monitor.MetricListInput)
	for _, measurement := range influxdbMeasurements {
		listInput.Measurement.Names = append(listInput.Measurement.Names, measurement.Measurement)
	}
	query, err := MetricMeasurementManager.ListItemFilter(context.Background(), MetricMeasurementManager.Query(), userCred,
		*listInput)
	if err != nil {
		log.Errorln(errors.Wrap(err, "DataSourceManager getMetricDescriptions error"))
	}
	descriMeasurements, err := MetricMeasurementManager.getMeasurement(query)
	if len(descriMeasurements) != 0 {

		measurementsIns := make([]interface{}, len(descriMeasurements))
		for i, _ := range descriMeasurements {
			measurementsIns[i] = &descriMeasurements[i]
		}
		details := MetricMeasurementManager.FetchCustomizeColumns(context.Background(), userCred, jsonutils.NewDict(), measurementsIns,
			stringutils2.NewSortedStrings([]string{}), true)
		if err != nil {
			log.Errorln(errors.Wrap(err, "DataSourceManager getMetricDescriptions error"))
		}
		for i, measureDes := range descriMeasurements {
			for j, _ := range influxdbMeasurements {
				if measureDes.Name == influxdbMeasurements[j].Measurement {
					if len(measureDes.DisplayName) != 0 {
						influxdbMeasurements[j].MeasurementDisplayName = measureDes.DisplayName
					}
					if len(measureDes.ResType) != 0 {
						influxdbMeasurements[j].ResType = measureDes.ResType
					}
					if measureDes.Score != 0 {
						influxdbMeasurements[j].Score = measureDes.Score
					}
					fieldDesMap := make(map[string]monitor.MetricFieldDetail, 0)
					fields := make([]string, 0)
					fieldKeys := stringutils2.NewSortedStrings(influxdbMeasurements[j].FieldKey)
					for fieldIndex, fieldDes := range details[i].MetricFields {
						if len(fieldDes.DisplayName) != 0 {
							fieldDesMap[fieldDes.Name] = details[i].MetricFields[fieldIndex]
						}
						if fieldKeys.Contains(fieldDes.Name) {
							fields = append(fields, fieldDes.Name)
						}
					}
					influxdbMeasurements[j].FieldDescriptions = fieldDesMap
					influxdbMeasurements[j].Database = measureDes.Database
					influxdbMeasurements[j].FieldKey = fields
					descMeasurements = append(descMeasurements, influxdbMeasurements[j])
				}
			}
		}
	}
	return
}

func (m *SDataSourceManager) filterMeasurementsByTime(
	measurements []monitor.InfluxMeasurement, query jsonutils.JSONObject, tagFilter *monitor.MetricQueryTag) ([]monitor.InfluxMeasurement, error) {
	timeF, err := m.getFromAndToFromParam(query)
	if err != nil {
		return nil, err
	}
	filterMeasurements, err := m.getFilterMeasurementsParallel(timeF.From, timeF.To, measurements, tagFilter)
	if err != nil {
		return nil, err
	}
	return filterMeasurements, nil
}

type timeFilter struct {
	From string
	To   string
}

func (m *SDataSourceManager) getFromAndToFromParam(query jsonutils.JSONObject) (timeFilter, error) {
	timeF := timeFilter{}
	from, _ := query.GetString("from")
	if len(from) == 0 {
		from = "6h"
	}
	to, _ := query.GetString("to")
	if len(to) == 0 {
		to = "now"
	}
	timeFilter := monitor.AlertQuery{
		From: from,
		To:   to,
	}
	err := validators.ValidateFromAndToValue(timeFilter)
	if err != nil {
		return timeF, err
	}
	timeF.From = from
	timeF.To = to
	return timeF, nil
}

func (m *SDataSourceManager) getFilterMeasurementsParallel(from, to string,
	measurements []monitor.InfluxMeasurement, tagFilter *monitor.MetricQueryTag) ([]monitor.InfluxMeasurement, error) {
	filterMeasurements := make([]monitor.InfluxMeasurement, len(measurements))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	measurementQueryGroup, _ := errgroup.WithContext(ctx)
	for i := range measurements {
		index := i
		tmp := measurements[index]
		measurementQueryGroup.Go(func() error {
			errCh := make(chan error)
			go func() {
				ret, err := m.getFilterMeasurement(from, to, tmp, tagFilter)
				if err != nil {
					errCh <- errors.Wrapf(err, "getFilterMeasurement %d", index)
					return
				}
				filterMeasurements[index] = *ret
				errCh <- nil
			}()

			for {
				select {
				case <-ctx.Done():
					return errors.Wrap(ctx.Err(), "filter measurement from TSDB")
				case err := <-errCh:
					if err != nil {
						return err
					}
					return nil
				}
			}
		})
	}
	if err := measurementQueryGroup.Wait(); err != nil {
		return nil, errors.Wrap(err, "measuremetnQueryGroup.Wait()")
	}
	ret := make([]monitor.InfluxMeasurement, 0)
	for _, fm := range filterMeasurements {
		if len(fm.Measurement) != 0 {
			tmp := fm
			ret = append(ret, tmp)
		}
	}
	return ret, nil
}

func (m *SDataSourceManager) GetTSDBDriver() (tsdb.TsdbQueryEndpoint, error) {
	ep, err := datasource.GetDefaultQueryEndpoint()
	if err != nil {
		return nil, errors.Wrap(err, "GetDefaultQueryEndpoint")
	}
	return ep, nil
}

func (m *SDataSourceManager) getFilterMeasurement(from, to string, measurement monitor.InfluxMeasurement, tagFilter *monitor.MetricQueryTag) (*monitor.InfluxMeasurement, error) {
	dds, _ := datasource.GetDefaultSource("")
	ep, err := m.GetTSDBDriver()
	if err != nil {
		return nil, errors.Wrap(err, "GetDefaultQueryEndpoint")
	}
	retMs, err := ep.FilterMeasurement(context.Background(), dds, from, to, &measurement, tagFilter)
	if err != nil {
		return nil, errors.Wrap(err, "Get endpoint filtered measurement")
	}
	return retMs, nil
}

func renderTimeFilter(from, to string) string {
	if strings.Contains(from, "now-") {
		from = "now() - " + strings.Replace(from, "now-", "", 1)
	} else {
		from = "now() - " + from
	}

	tmp := ""
	if to != "now" && to != "" {
		tmp = " and time < now() - " + strings.Replace(to, "now-", "", 1)
	}

	return fmt.Sprintf("time > %s%s", from, tmp)

}

func (m *SDataSourceManager) GetMetricMeasurement(userCred mcclient.TokenCredential, query jsonutils.JSONObject, tagFilter *monitor.MetricQueryTag) (jsonutils.JSONObject, error) {
	database, _ := query.GetString("database")
	if database == "" {
		return jsonutils.JSONNull, merrors.NewArgIsEmptyErr("database")
	}
	measurement, _ := query.GetString("measurement")
	if measurement == "" {
		return jsonutils.JSONNull, merrors.NewArgIsEmptyErr("measurement")
	}
	field, _ := query.GetString("field")
	if field == "" {
		return jsonutils.JSONNull, merrors.NewArgIsEmptyErr("field")
	}
	from, _ := query.GetString("from")
	if len(from) == 0 {
		return jsonutils.JSONNull, merrors.NewArgIsEmptyErr("from")
	}
	timeF, err := m.getFromAndToFromParam(query)
	if err != nil {
		return nil, errors.Wrap(err, "getFromAndToFromParam")
	}

	skipCheckSeries := jsonutils.QueryBoolean(query, "skip_check_series", false)

	output := new(monitor.InfluxMeasurement)
	output.Measurement = measurement
	output.Database = database
	output.TagValue = make(map[string][]string, 0)

	output.FieldKey = []string{field}
	if err := getTagValues(userCred, output, timeF, tagFilter, skipCheckSeries); err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "getTagValues error")
	}

	m.filterRtnTags(output)
	return jsonutils.Marshal(output), nil

}

func (m *SDataSourceManager) filterRtnTags(output *monitor.InfluxMeasurement) {
	for _, tag := range []string{hostconsts.TELEGRAF_TAG_KEY_BRAND, hostconsts.TELEGRAF_TAG_KEY_PLATFORM,
		hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR} {
		if val, ok := output.TagValue[tag]; ok {
			output.TagValue[hostconsts.TELEGRAF_TAG_KEY_BRAND] = val
			break
		}
	}
	for _, tag := range []string{
		"source", "status", hostconsts.TELEGRAF_TAG_KEY_HOST_TYPE, hostconsts.TELEGRAF_TAG_KEY_RES_TYPE,
		"is_vm", "os_type", hostconsts.TELEGRAF_TAG_KEY_PLATFORM, hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR,
		"domain_name", "region", "ips", "vip", "vip_eip", "eip", "eip_mode",
		labels.MetricName, translator.UNION_RESULT_NAME,
	} {
		if _, ok := output.TagValue[tag]; ok {
			delete(output.TagValue, tag)
		}
	}
	// hide VictoriaMetrics telegraf db tag
	if val, ok := output.TagValue[VICTORIA_METRICS_DB_TAG_KEY]; ok {
		if len(val) == 1 && val[0] == VICTORIA_METRICS_DB_TAG_VAL_TELEGRAF {
			delete(output.TagValue, VICTORIA_METRICS_DB_TAG_KEY)
		}
	}

	repTag := make([]string, 0)
	for tag, _ := range output.TagValue {
		repTag = append(repTag, tag)
	}
	output.TagKey = repTag
}

func (m *SDataSourceManager) filterTagValue(measurement monitor.InfluxMeasurement, timeF timeFilter,
	db *influxdb.SInfluxdb, tagValChan *influxdbTagValueChan, tagFilter string) error {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*15)
	tagValGroup2, _ := errgroup.WithContext(ctx)
	tagValChan2 := influxdbTagValueChan{
		rtnChan: make(chan map[string][]string, len(measurement.TagKey)),
		count:   len(measurement.TagKey),
	}
	for i, _ := range measurement.TagKey {
		tmpkey := measurement.TagKey[i]
		tagValGroup2.Go(func() error {
			return m.getFilterMeasurementTagValue(&tagValChan2, timeF.From, timeF.To, measurement.FieldKey[0],
				tmpkey, measurement, db, tagFilter)
		})
	}
	tagValGroup2.Go(func() error {
		valMaps := make(map[string][]string, 0)
		for i := 0; i < tagValChan2.count; i++ {
			select {
			case valMap := <-tagValChan2.rtnChan:
				for key, val := range valMap {
					if _, ok := valMaps[key]; ok {
						valMaps[key] = append(valMaps[key], val...)
						continue
					}
					valMaps[key] = val
				}
			case <-ctx.Done():
				return fmt.Errorf("filter getFilterMeasurementTagValue time out")
			}
		}
		tagValChan.rtnChan <- valMaps
		close(tagValChan2.rtnChan)
		return nil
	})
	return tagValGroup2.Wait()
}

func tagValUnion(measurement *monitor.InfluxMeasurement, rtn map[string][]string) {
	keys := make([]string, 0)
	for _, tag := range measurement.TagKey {
		if rtnTagVal, ok := rtn[tag]; ok {
			keys = append(keys, tag)
			if _, ok := measurement.TagValue[tag]; !ok {
				measurement.TagValue[tag] = rtnTagVal
				continue
			}
			measurement.TagValue[tag] = union(measurement.TagValue[tag], rtnTagVal)
		}
	}
	measurement.TagKey = keys
}

func union(slice1, slice2 []string) []string {
	m := make(map[string]int)
	for _, v := range slice1 {
		m[v]++
	}

	for _, v := range slice2 {
		times, _ := m[v]
		if times == 0 {
			slice1 = append(slice1, v)
		}
	}
	return slice1
}

type InfluxdbSubscription struct {
	SubName  string
	DataBase string
	//retention policy
	Rc  string
	Url string
}

func (m *SDataSourceManager) AddSubscription(subscription InfluxdbSubscription) error {

	query := fmt.Sprintf("CREATE SUBSCRIPTION %s ON %s.%s DESTINATIONS ALL %s",
		jsonutils.NewString(subscription.SubName).String(),
		jsonutils.NewString(subscription.DataBase).String(),
		jsonutils.NewString(subscription.Rc).String(),
		strings.ReplaceAll(jsonutils.NewString(subscription.Url).String(), "\"", "'"),
	)
	dataSource, err := datasource.GetDefaultSource("")
	if err != nil {
		return errors.Wrap(err, "s.GetDefaultSource")
	}

	db := influxdb.NewInfluxdbWithDebug(dataSource.Url, true)
	db.SetDatabase(subscription.DataBase)

	rtn, err := db.GetQuery(query)
	if err != nil {
		return err
	}
	for _, result := range rtn {
		for _, obj := range result {
			objJson := jsonutils.Marshal(&obj)
			log.Errorln(objJson.String())
		}
	}
	return nil
}

func (m *SDataSourceManager) DropSubscription(subscription InfluxdbSubscription) error {
	query := fmt.Sprintf("DROP SUBSCRIPTION %s ON %s.%s", jsonutils.NewString(subscription.SubName).String(),
		jsonutils.NewString(subscription.DataBase).String(),
		jsonutils.NewString(subscription.Rc).String(),
	)
	dataSource, err := datasource.GetDefaultSource("")
	if err != nil {
		return errors.Wrap(err, "s.GetDefaultSource")
	}

	db := influxdb.NewInfluxdb(dataSource.Url)
	db.SetDatabase(subscription.DataBase)
	rtn, err := db.Query(query)
	if err != nil {
		return err
	}
	for _, result := range rtn {
		for _, obj := range result {
			objJson := jsonutils.Marshal(&obj)
			log.Errorln(objJson.String())
		}
	}
	return nil
}

func getAttributesOnMeasurement(database, tp string, output *monitor.InfluxMeasurement, db *influxdb.SInfluxdb) error {
	query := fmt.Sprintf("SHOW %s KEYS ON %s FROM %s", tp, database, output.Measurement)
	dbRtn, err := db.Query(query)
	if err != nil {
		return errors.Wrapf(err, "SHOW MEASUREMENTS: %s", query)
	}
	if len(dbRtn) == 0 || len(dbRtn[0]) == 0 {
		return nil
	}
	res := dbRtn[0][0]
	tmpDict := jsonutils.NewDict()
	tmpArr := jsonutils.NewArray()
	for i := range res.Values {
		v, _ := res.Values[i][0].(*jsonutils.JSONString).GetString()
		if filterTagKey(v) {
			continue
		}
		tmpArr.Add(res.Values[i][0])
	}
	tmpDict.Add(tmpArr, res.Columns[0])
	err = tmpDict.Unmarshal(output)
	if err != nil {
		return errors.Wrap(err, "measurement unmarshal error")
	}
	return nil
}

func getTagValues(userCred mcclient.TokenCredential, output *monitor.InfluxMeasurement, timeF timeFilter, tagFilter *monitor.MetricQueryTag, skipCheckSeries bool) error {
	mq := monitor.MetricQuery{
		Database:    output.Database,
		Measurement: output.Measurement,
		Selects: []monitor.MetricQuerySelect{
			{
				{
					Type:   "field",
					Params: []string{output.FieldKey[0]},
				},
				{
					Type: "last",
				},
			},
		},
		GroupBy: []monitor.MetricQueryPart{
			{
				Type:   "field",
				Params: []string{"*"},
			},
		},
	}
	if tagFilter != nil {
		mq.Tags = []monitor.MetricQueryTag{
			{
				Key:      tagFilter.Key,
				Operator: tagFilter.Operator,
				Value:    tagFilter.Value,
			},
		}
	}

	aq := &monitor.AlertQuery{
		Model: mq,
		From:  timeF.From,
		To:    timeF.To,
	}

	q := monitor.MetricQueryInput{
		From:     timeF.From,
		To:       timeF.To,
		Interval: "5m",
		MetricQuery: []*monitor.AlertQuery{
			aq,
		},
		SkipCheckSeries: skipCheckSeries,
	}

	ret, err := doQuery(userCred, q)
	if err != nil {
		return errors.Wrapf(err, "getTagValues query error %s", jsonutils.Marshal(q))
	}

	// 2. group tag and values
	tagValMap := make(map[string][]string)
	tagKeys := make([]string, 0)
	if len(ret.Series) == 0 {
		return nil
	}

	for _, s := range ret.Series {
		tagMap := s.Tags
		for key, valStr := range tagMap {
			valStr = renderTagVal(valStr)
			if len(valStr) == 0 || valStr == "null" || filterTagValue(valStr) {
				continue
			}
			if filterTagKey(key) {
				continue
			}
			if valArr, ok := tagValMap[key]; ok {
				if !utils.IsInStringArray(valStr, valArr) {
					tagValMap[key] = append(valArr, valStr)
				}
				continue
			}
			tagValMap[key] = []string{valStr}
			tagKeys = append(tagKeys, key)
		}
	}
	output.TagValue = tagValMap
	sort.Strings(tagKeys)
	output.TagKey = tagKeys

	return nil
}

func getTagValue(database string, output *monitor.InfluxMeasurement, db *influxdb.SInfluxdb) error {
	if len(output.TagKey) == 0 {
		return nil
	}
	tagKeyStr := jsonutils.NewStringArray(output.TagKey).String()
	tagKeyStr = tagKeyStr[1 : len(tagKeyStr)-1]
	dbRtn, err := db.Query(fmt.Sprintf("SHOW TAG VALUES ON %s FROM %s WITH KEY IN (%s)", database, output.Measurement, tagKeyStr))
	if err != nil {
		return err
	}
	res := dbRtn[0][0]
	tagValue := make(map[string][]string, 0)
	keys := strings.Join(output.TagKey, ",")
	for i := range res.Values {
		val, _ := res.Values[i][0].(*jsonutils.JSONString).GetString()
		if !strings.Contains(keys, val) {
			continue
		}
		if _, ok := tagValue[val]; !ok {
			tagValue[val] = make([]string, 0)
		}
		tag, _ := res.Values[i][1].(*jsonutils.JSONString).GetString()
		if filterTagValue(tag) {
			delete(tagValue, val)
			continue
		}
		tagValue[val] = append(tagValue[val], tag)
	}
	output.TagValue = tagValue
	//TagKey == TagValue.keys
	tagK := make([]string, 0)
	for tag, _ := range output.TagValue {
		tagK = append(tagK, tag)
	}
	output.TagKey = tagK
	return nil
}

type influxdbTagValueChan struct {
	rtnChan chan map[string][]string
	count   int
}

func (m *SDataSourceManager) getFilterMeasurementTagValue(tagValueChan *influxdbTagValueChan, from string,
	to string, field string, tagKey string,
	measurement monitor.InfluxMeasurement, db *influxdb.SInfluxdb, tagFilter string) error {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf(`SELECT last("%s") FROM "%s" WHERE %s `, field, measurement.Measurement,
		renderTimeFilter(from, to)))
	if len(tagFilter) != 0 {
		buffer.WriteString(fmt.Sprintf(` AND %s `, tagFilter))
	}
	buffer.WriteString(fmt.Sprintf(` GROUP BY %q`, tagKey))
	log.Errorln(buffer.String())
	rtn, err := db.Query(buffer.String())
	if err != nil {
		return errors.Wrap(err, "getFilterMeasurementTagValue query error")
	}
	tagValMap := make(map[string][]string)
	if len(rtn) != 0 && len(rtn[0]) != 0 {
		for rtnIndex, _ := range rtn {
			for serieIndex, _ := range rtn[rtnIndex] {
				tagMap, _ := rtn[rtnIndex][serieIndex].Tags.GetMap()
				for key, valObj := range tagMap {
					valStr, _ := valObj.GetString()
					valStr = renderTagVal(valStr)
					if len(valStr) == 0 || valStr == "null" || filterTagValue(valStr) {
						continue
					}
					if !utils.IsInStringArray(key, measurement.TagKey) {
						//measurement.TagKey = append(measurement.TagKey, key)
						continue
					}
					if valArr, ok := tagValMap[key]; ok {
						if !utils.IsInStringArray(valStr, valArr) {
							tagValMap[key] = append(valArr, valStr)
						}
						continue
					}
					tagValMap[key] = []string{valStr}
				}
			}
		}
		measurement.TagValue = tagValMap
	}
	tagValueChan.rtnChan <- tagValMap
	return nil
}

func renderTagVal(val string) string {
	return strings.ReplaceAll(val, "+", " ")
}
func floatEquals(a, b float64) bool {
	eps := 0.000000001
	if math.Abs(a-b) < eps {
		return true
	}
	return false
}

var filterKey = []string{"perf_instance", "res_type", "status", "cloudregion", "os_type", "is_vm"}

func filterTagKey(key string) bool {
	whiteListIdKeys := sets.NewString("dev_id", "die_id")
	if whiteListIdKeys.Has(key) {
		return false
	}
	if strings.Contains(key, "_id") {
		return true
	}
	if key == "perf_instance" {
		return true
	}
	return false
}

func filterTagValue(val string) bool {
	if compile.MatchString(val) {
		return true
	}
	return false
}
