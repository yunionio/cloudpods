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
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/wait"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	merrors "yunion.io/x/onecloud/pkg/monitor/errors"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/registry"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/monitor/validators"
	"yunion.io/x/onecloud/pkg/util/influxdb"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	DataSourceManager *SDataSourceManager
	compile           = regexp.MustCompile(`\w{8}(-\w{4}){3}-\w{12}`)
)

const (
	DefaultDataSource = "default"
)

const (
	ErrDataSourceDefaultNotFound = errors.Error("Default data source not found")
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
	registry.RegisterService(DataSourceManager)
}

type SDataSourceManager struct {
	db.SStandaloneResourceBaseManager
}

func (_ *SDataSourceManager) IsDisabled() bool {
	return false
}

func (_ *SDataSourceManager) Init() error {
	return nil
}

func (man *SDataSourceManager) Run(ctx context.Context) error {
	errgrp, ctx := errgroup.WithContext(ctx)
	errgrp.Go(func() error { return man.initDefaultDataSource(ctx) })
	return errgrp.Wait()
}

func (man *SDataSourceManager) initDefaultDataSource(ctx context.Context) error {
	region := options.Options.Region
	epType := options.Options.SessionEndpointType
	initF := func() {
		ds, err := man.GetDefaultSource()
		if err != nil && err != ErrDataSourceDefaultNotFound {
			log.Errorf("Get default datasource: %v", err)
			return
		}
		s := auth.GetAdminSession(ctx, region)
		if s == nil {
			log.Errorf("get empty public session for region %s", region)
			return
		}
		url, err := s.GetServiceURL("influxdb", epType)
		if err != nil {
			log.Errorf("get influxdb public url: %v", err)
			return
		}
		if ds != nil {
			if _, err := db.Update(ds, func() error {
				ds.Url = url
				return nil
			}); err != nil {
				log.Errorf("update datasource url error: %v", err)
			}
			return
		}
		ds = &SDataSource{
			Type: monitor.DataSourceTypeInfluxdb,
			Url:  url,
		}
		ds.Name = DefaultDataSource
		if err := man.TableSpec().Insert(ctx, ds); err != nil {
			log.Errorf("insert default influxdb: %v", err)
		}
	}
	wait.Forever(initF, 30*time.Second)
	return nil
}

func (man *SDataSourceManager) GetDefaultSource() (*SDataSource, error) {
	obj, err := man.FetchByName(nil, DefaultDataSource)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDataSourceDefaultNotFound
		} else {
			return nil, err
		}
	}
	return obj.(*SDataSource), nil
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

func (ds *SDataSource) ToTSDBDataSource(db string) *tsdb.DataSource {
	if db == "" {
		db = ds.Database
	}
	return &tsdb.DataSource{
		Id:       ds.GetId(),
		Name:     ds.GetName(),
		Type:     ds.Type,
		Url:      ds.Url,
		User:     ds.User,
		Password: ds.Password,
		Database: db,
		Updated:  ds.UpdatedAt,
		/*BasicAuth: ds.BasicAuth,
		BasicAuthUser: ds.BasicAuthUser,
		BasicAuthPassword: ds.BasicAuthPassword,
		TimeInterval: ds.TimeInterval,*/
	}
}

func (self *SDataSourceManager) GetDatabases() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	dataSource, err := self.GetDefaultSource()
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

func (self *SDataSourceManager) GetMeasurements(query jsonutils.JSONObject,
	measurementFilter, tagFilter string) (jsonutils.JSONObject,
	error) {
	ret := jsonutils.NewDict()
	measurements, err := self.getMeasurementQueryInfluxdb(query, measurementFilter, tagFilter)
	if err != nil {
		return jsonutils.JSONNull, err
	}
	ret.Add(jsonutils.Marshal(&measurements), "measurements")
	return ret, nil
}

func (self *SDataSourceManager) getMeasurementQueryInfluxdb(query jsonutils.JSONObject,
	measurementFilter, tagFilter string) (rtnMeasurements []monitor.InfluxMeasurement, err error) {
	database, _ := query.GetString("database")
	if database == "" {
		return rtnMeasurements, merrors.NewArgIsEmptyErr("database")
	}
	dataSource, err := self.GetDefaultSource()
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

func (self *SDataSourceManager) GetMeasurementsWithDescriptionInfos(query jsonutils.JSONObject, measurementFilter,
	tagFilter string) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	rtnMeasurements := make([]monitor.InfluxMeasurement, 0)
	measurements, err := MetricMeasurementManager.getInfluxdbMeasurements()
	if err != nil {
		return jsonutils.JSONNull, err
	}
	dataSource, err := self.GetDefaultSource()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetDefaultSource")
	}
	db := influxdb.NewInfluxdb(dataSource.Url)
	filterMeasurements, err := self.filterMeasurementsByTime(*db, measurements, query, tagFilter)
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "filterMeasurementsByTime error")
	}
	filterMeasurements = self.getMetricDescriptions(filterMeasurements)
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

func (self *SDataSourceManager) GetMeasurementsWithOutTimeFilter(query jsonutils.JSONObject,
	measurementFilter, tagFilter string) (jsonutils.JSONObject,
	error) {
	ret := jsonutils.NewDict()
	database, _ := query.GetString("database")
	if database == "" {
		return jsonutils.JSONNull, httperrors.NewInputParameterError("not support database")
	}
	dataSource, err := self.GetDefaultSource()
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

func (self *SDataSourceManager) getMetricDescriptions(influxdbMeasurements []monitor.InfluxMeasurement) (
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

type influxdbQueryChan struct {
	queryRtnChan chan monitor.InfluxMeasurement
	count        int
}

func (self *SDataSourceManager) filterMeasurementsByTime(db influxdb.SInfluxdb,
	measurements []monitor.InfluxMeasurement, query jsonutils.JSONObject, tagFilter string) ([]monitor.InfluxMeasurement,
	error) {
	timeF, err := self.getFromAndToFromParam(query)
	if err != nil {
		return nil, err
	}
	filterMeasurements, err := self.getFilterMeasurementsAsyn(timeF.From, timeF.To, measurements, db, tagFilter)
	if err != nil {
		return nil, err
	}
	return filterMeasurements, nil
}

type timeFilter struct {
	From string
	To   string
}

func (self *SDataSourceManager) getFromAndToFromParam(query jsonutils.JSONObject) (timeFilter, error) {
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

func (self *SDataSourceManager) getFilterMeasurementsAsyn(from, to string,
	measurements []monitor.InfluxMeasurement, db influxdb.SInfluxdb, tagFilter string) ([]monitor.InfluxMeasurement, error) {
	filterMeasurements := make([]monitor.InfluxMeasurement, 0)
	queryChan := new(influxdbQueryChan)
	queryChan.queryRtnChan = make(chan monitor.InfluxMeasurement, len(measurements))
	queryChan.count = len(measurements)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	measurementQueryGroup, _ := errgroup.WithContext(ctx)
	for i, _ := range measurements {
		tmp := measurements[i]
		measurementQueryGroup.Go(func() error {
			return self.getFilterMeasurement(queryChan, from, to, tmp, db, tagFilter)
		})
	}
	measurementQueryGroup.Go(func() error {
		for i := 0; i < queryChan.count; i++ {
			select {
			case filterMeasurement := <-queryChan.queryRtnChan:
				if len(filterMeasurement.Measurement) != 0 {
					filterMeasurements = append(filterMeasurements, filterMeasurement)
				}
			case <-ctx.Done():
				return fmt.Errorf("filter measurement time out")
			}
		}
		return nil
	})
	err := measurementQueryGroup.Wait()
	return filterMeasurements, err
}

func (self *SDataSourceManager) getFilterMeasurement(queryChan *influxdbQueryChan, from, to string,
	measurement monitor.InfluxMeasurement, db influxdb.SInfluxdb, tagFilter string) error {
	rtnMeasurement := new(monitor.InfluxMeasurement)
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf(`SELECT last(*) FROM %s WHERE %s`, measurement.Measurement,
		renderTimeFilter(from, to)))
	if len(tagFilter) != 0 {
		buffer.WriteString(" AND ")
		buffer.WriteString(fmt.Sprintf(" %s", tagFilter))
	}
	log.Errorln(buffer.String())
	(&db).SetDatabase(measurement.Database)
	rtn, err := db.Query(buffer.String())
	if err != nil {
		return errors.Wrap(err, "getFilterMeasurement error")
	}

	rtnFields := make([]string, 0)
	if len(rtn) != 0 && len(rtn[0]) != 0 {
		for rtnIndex, _ := range rtn {
			for serieIndex, _ := range rtn[rtnIndex] {
				meanFieldArr := rtn[rtnIndex][serieIndex].Columns
				for i, _ := range meanFieldArr {
					if !strings.Contains(meanFieldArr[i], "last") {
						continue
					}

					containsVal := false
					for _, value := range rtn[rtnIndex][serieIndex].Values {
						if value[i] == nil {
							continue
						}
						_, err := value[i].Float()
						if err != nil {
							continue
						}
						containsVal = true
						break
					}
					if containsVal {
						rtnFields = append(rtnFields, strings.Replace(meanFieldArr[i], "last_", "", 1))
					}
				}
			}
		}
	}
	rtnMeasurement.FieldKey = rtnFields
	if len(rtnMeasurement.FieldKey) != 0 {
		rtnMeasurement.Measurement = measurement.Measurement
		rtnMeasurement.Database = measurement.Database
		rtnMeasurement.ResType = measurement.ResType
	}
	queryChan.queryRtnChan <- *rtnMeasurement
	return nil
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

func (self *SDataSourceManager) GetMetricMeasurement(userCred mcclient.TokenCredential, query jsonutils.JSONObject, tagFilter string) (jsonutils.JSONObject, error) {
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
	dataSource, err := self.GetDefaultSource()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetDefaultSource")
	}

	timeF, err := self.getFromAndToFromParam(query)
	if err != nil {
		return nil, errors.Wrap(err, "getFromAndToFromParam")
	}

	skipCheckSeries := jsonutils.QueryBoolean(query, "skip_check_series", false)

	db := influxdb.NewInfluxdb(dataSource.Url)
	db.SetDatabase(database)

	output := new(monitor.InfluxMeasurement)
	output.Measurement = measurement
	output.Database = database
	output.TagValue = make(map[string][]string, 0)
	// for _, val := range monitor.METRIC_ATTRI {
	// 	if err := getAttributesOnMeasurement(database, val, output, db); err != nil {
	// 		return jsonutils.JSONNull, errors.Wrap(err, "getAttributesOnMeasurement error")
	// 	}
	// }

	output.FieldKey = []string{field}
	//err = getTagValue(database, output, db)

	// tagValChan := influxdbTagValueChan{
	// 	rtnChan: make(chan map[string][]string, len(output.FieldKey)),
	// 	count:   len(output.FieldKey),
	// 	//count: 1,
	// }

	//** ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	// tagValGroup, _ := errgroup.WithContext(ctx)
	// defer cancel()
	// tagValGroup.Go(func() error {
	// 	return self.filterTagValue(*output, timeF, db, &tagValChan, tagFilter)
	// })
	// tagValGroup.Go(func() error {
	// 	for i := 0; i < tagValChan.count; i++ {
	// 		select {
	// 		case tagVal := <-tagValChan.rtnChan:
	// 			if len(tagVal) != 0 {
	// 				tagValUnion(output, tagVal)
	// 			}
	// 		case <-ctx.Done():
	// 			return fmt.Errorf("filter Union TagValue time out")
	// 		}
	// 	}
	// 	return nil
	// })
	// err = tagValGroup.Wait()
	// if err != nil {
	// 	return jsonutils.JSONNull, errors.Wrap(err, "getTagValue error")
	//** }
	if err := getTagValues(userCred, output, timeF, dataSource.GetId(), tagFilter, skipCheckSeries); err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "getTagValues error")
	}

	self.filterRtnTags(output)
	return jsonutils.Marshal(output), nil

}

func (self *SDataSourceManager) filterRtnTags(output *monitor.InfluxMeasurement) {
	for _, tag := range []string{hostconsts.TELEGRAF_TAG_KEY_BRAND, hostconsts.TELEGRAF_TAG_KEY_PLATFORM,
		hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR} {
		if val, ok := output.TagValue[tag]; ok {
			output.TagValue[hostconsts.TELEGRAF_TAG_KEY_BRAND] = val
			break
		}
	}
	for _, tag := range []string{"source", "status", hostconsts.TELEGRAF_TAG_KEY_HOST_TYPE,
		hostconsts.TELEGRAF_TAG_KEY_RES_TYPE, "is_vm", "os_type", hostconsts.TELEGRAF_TAG_KEY_PLATFORM,
		hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR, "domain_name", "region", "ips", "vip", "vip_eip", "eip", "eip_mode"} {
		if _, ok := output.TagValue[tag]; ok {
			delete(output.TagValue, tag)
		}
	}

	repTag := make([]string, 0)
	for tag, _ := range output.TagValue {
		repTag = append(repTag, tag)
	}
	output.TagKey = repTag
}

func (self *SDataSourceManager) filterTagValue(measurement monitor.InfluxMeasurement, timeF timeFilter,
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
			return self.getFilterMeasurementTagValue(&tagValChan2, timeF.From, timeF.To, measurement.FieldKey[0],
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

func (self *SDataSourceManager) AddSubscription(subscription InfluxdbSubscription) error {

	query := fmt.Sprintf("CREATE SUBSCRIPTION %s ON %s.%s DESTINATIONS ALL %s",
		jsonutils.NewString(subscription.SubName).String(),
		jsonutils.NewString(subscription.DataBase).String(),
		jsonutils.NewString(subscription.Rc).String(),
		strings.ReplaceAll(jsonutils.NewString(subscription.Url).String(), "\"", "'"),
	)
	dataSource, err := self.GetDefaultSource()
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

func (self *SDataSourceManager) DropSubscription(subscription InfluxdbSubscription) error {
	query := fmt.Sprintf("DROP SUBSCRIPTION %s ON %s.%s", jsonutils.NewString(subscription.SubName).String(),
		jsonutils.NewString(subscription.DataBase).String(),
		jsonutils.NewString(subscription.Rc).String(),
	)
	dataSource, err := self.GetDefaultSource()
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

func getTagValues(userCred mcclient.TokenCredential, output *monitor.InfluxMeasurement, timeF timeFilter, dsId string, tagFilter string, skipCheckSeries bool) error {
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
	if tagFilter != "" {
		parts := strings.Split(tagFilter, " ")
		mq.Tags = []monitor.MetricQueryTag{
			{
				Key:      parts[0],
				Operator: parts[1],
				Value:    parts[2],
			},
		}
	}

	aq := &monitor.AlertQuery{
		Model:        mq,
		From:         timeF.From,
		To:           timeF.To,
		DataSourceId: dsId,
	}

	q := monitor.MetricInputQuery{
		From: timeF.From,
		To:   timeF.To,
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

func (self *SDataSourceManager) getFilterMeasurementTagValue(tagValueChan *influxdbTagValueChan, from string,
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
