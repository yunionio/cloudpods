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
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/wait"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/monitor/registry"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

var (
	DataSourceManager *SDataSourceManager
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
	initF := func() {
		ds, err := man.GetDefaultSource()
		if err != nil && err != ErrDataSourceDefaultNotFound {
			log.Errorf("Get default datasource: %v", err)
			return
		}
		if ds != nil {
			return
		}
		s := auth.GetAdminSessionWithPublic(ctx, region, "")
		if s == nil {
			log.Errorf("get empty public session for region %s", region)
			return
		}
		url, err := s.GetServiceURL("influxdb", identityapi.EndpointInterfacePublic)
		if err != nil {
			log.Errorf("get influxdb public url: %v", err)
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
	IsDefault tristate.TriState `nullable:"false" default:"false" create:"optional"`
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

func (self *SDataSourceManager) GetMeasurements(query jsonutils.JSONObject, filter string) (jsonutils.JSONObject,
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
	var q string
	if filter != "" {
		q = fmt.Sprintf("SHOW MEASUREMENTS ON %s WHERE %s", database, filter)
	} else {
		q = fmt.Sprintf("SHOW MEASUREMENTS ON %s", database)
	}

	dbRtn, err := db.Query(q)
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "SHOW MEASUREMENTS")
	}
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
	return ret, nil
}

func (self *SDataSourceManager) GetMetricMeasurement(query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	database, _ := query.GetString("database")
	if database == "" {
		return jsonutils.JSONNull, httperrors.NewInputParameterError("not support database")
	}
	measurement, _ := query.GetString("measurement")
	if measurement == "" {
		return jsonutils.JSONNull, httperrors.NewInputParameterError("not support measurement")
	}
	dataSource, err := self.GetDefaultSource()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetDefaultSource")
	}

	db := influxdb.NewInfluxdb(dataSource.Url)
	db.SetDatabase(database)
	output := new(monitor.InfluxMeasurement)
	output.Measurement = measurement
	output.Database = database
	for _, val := range monitor.METRIC_ATTRI {
		err = getAttributesOnMeasurement(database, val, output, db)
		if err != nil {
			return jsonutils.JSONNull, errors.Wrap(err, "getAttributesOnMeasurement error")
		}
	}
	err = getTagValue(database, output, db)
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "getTagValue error")
	}
	return jsonutils.Marshal(output), nil

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
	dbRtn, err := db.Query(fmt.Sprintf("SHOW %s KEYS ON %s FROM %s", tp, database, output.Measurement))
	if err != nil {
		return errors.Wrap(err, "SHOW MEASUREMENTS")
	}
	res := dbRtn[0][0]
	tmpDict := jsonutils.NewDict()
	tmpArr := jsonutils.NewArray()
	for i := range res.Values {
		tmpArr.Add(res.Values[i][0])
	}
	tmpDict.Add(tmpArr, res.Columns[0])
	err = tmpDict.Unmarshal(output)
	if err != nil {
		return errors.Wrap(err, "measurement unmarshal error")
	}
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
	for i := range res.Values {
		val := res.Values[i][0].(*jsonutils.JSONString)
		if _, ok := tagValue[val.Value()]; !ok {
			tagValue[val.Value()] = make([]string, 0)
		}
		tag := res.Values[i][1].(*jsonutils.JSONString)
		tagValue[val.Value()] = append(tagValue[val.Value()], tag.Value())
	}
	output.TagValue = tagValue
	return nil
}
