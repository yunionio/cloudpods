package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

type InfluxdbShemaListOptions struct {
}

type InfluxdbShemaShowOptions struct {
	ID          string `help:"attribute of the inluxdb" choices:"databases|measurements|metric-measurement"`
	Database    string `help:"influxdb database"`
	Measurement string `help:"influxdb table"`
}

func (opt InfluxdbShemaShowOptions) Params() (jsonutils.JSONObject, error) {
	input := new(monitor.InfluxMeasurement)
	input.Measurement = opt.Measurement
	input.Database = opt.Database
	return input.JSON(input), nil
}
