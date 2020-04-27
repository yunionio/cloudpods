package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	. "yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	o "yunion.io/x/onecloud/pkg/monitor/options"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

var (
	InfluxdbSchemaManager *SInfluxdbSchemaManager
)

func init() {
	InfluxdbSchemaManager = &SInfluxdbSchemaManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&SInfluxdbSchemaManager{},
			"",
			"influxdbshema",
			"influxdbshemas",
		),
	}
	InfluxdbSchemaManager.SetVirtualObject(InfluxdbSchemaManager)
}

type SInfluxdbSchemaManager struct {
	db.SVirtualResourceBaseManager
}

type SInfluxdbSchemaModel struct {
}

func (self *SInfluxdbSchemaManager) AllowGetPropertyDatabases(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (self *SInfluxdbSchemaManager) AllowGetPropertyMeasurements(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (self *SInfluxdbSchemaManager) AllowGetPropertyMetricMeasurement(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (self *SInfluxdbSchemaManager) GetPropertyDatabases(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	url, err := getInfluxdbUrl()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetServiceURLs")
	}
	db := influxdb.NewInfluxdb(url)
	//db.SetDatabase("telegraf")
	databases, err := db.GetDatabases()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "GetDatabases")
	}
	ret.Add(jsonutils.NewStringArray(databases), "databases")
	return ret, nil
}

func (self *SInfluxdbSchemaManager) GetPropertyMeasurements(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	database, _ := query.GetString("database")
	if database == "" {
		return jsonutils.JSONNull, NewInputParameterError("not support database")
	}
	url, err := getInfluxdbUrl()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetServiceURLs")
	}
	db := influxdb.NewInfluxdb(url)
	db.SetDatabase(database)
	dbRtn, err := db.Query(fmt.Sprintf("SHOW MEASUREMENTS ON %s", database))
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

func (self *SInfluxdbSchemaManager) GetPropertyMetricMeasurement(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	database, _ := query.GetString("database")
	if database == "" {
		return jsonutils.JSONNull, NewInputParameterError("not support database")
	}
	measurement, _ := query.GetString("measurement")
	if measurement == "" {
		return jsonutils.JSONNull, NewInputParameterError("not support measurement")
	}
	url, err := getInfluxdbUrl()
	if err != nil {
		return jsonutils.JSONNull, errors.Wrap(err, "s.GetServiceURLs")
	}
	db := influxdb.NewInfluxdb(url)
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
	return jsonutils.Marshal(output), nil

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

func getInfluxdbUrl() (string, error) {
	s := auth.GetAdminSession(context.Background(), consts.GetRegion(), "")
	return s.GetServiceURL("influxdb", o.Options.SessionEndpointType)
}
