package monitor

import (
	monitorapi "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type MonitorMetricListOptions struct {
		options.BaseListOptions
		MeasurementName        []string `help:"name of Measurement"`
		ResType                string   `help:"Resource properties of measurement e.g. guest/host/redis/oss/rds"`
		MeasurementDisplayName string   `help:"The name of the measurement customization"`
		FieldName              []string `help:"Name of Field"`
		Unit                   string   `help:"Unit of Field " choices:"%|bps|Mbps|Bps|cps|count|ms|byte"`
		FieldDisplayName       string   `help:"The name of the field customization"`
	}
	R(&MonitorMetricListOptions{}, "metric-describ-list", "list metric description info", func(s *mcclient.ClientSession,
		args *MonitorMetricListOptions) error {
		param, err := options.ListStructToParams(&(args.BaseListOptions))
		if err != nil {
			return err
		}
		metricInput := new(monitorapi.MetricListInput)
		metricInput.Measurement.Names = args.MeasurementName
		metricInput.Measurement.ResType = args.ResType
		metricInput.Measurement.DisplayName = args.MeasurementDisplayName
		metricInput.MetricFields.Names = args.FieldName
		metricInput.MetricFields.Unit = args.Unit
		metricInput.MetricFields.DisplayName = args.FieldDisplayName
		listParam := metricInput.JSON(metricInput)
		param.Update(listParam)
		result, err := modules.MetricManager.List(s, param)
		if err != nil {
			return err
		}
		printList(result, modules.MetricManager.GetColumns(s))
		return nil
	})

	type MetricUpdateOptions struct {
		ID                     string `help:"ID of Metric " required:"true" positional:"true"`
		ResType                string `help:"Resource properties of measurement e.g. guest/host/redis/oss/rds" required:"true"`
		MeasurementDisplayName string `help:"The name of the measurement customization" required:"true"`
		FieldName              string `help:"Name of Field" required:"true"`
		FieldDisplayName       string `help:"The name of the field customization" required:"true"`
		Unit                   string `help:"Unit of Field" choices:"%|bps|Mbps|Bps|cps|count|ms|byte" required:"true"`
	}
	R(&MetricUpdateOptions{}, "metric-describ-update", "update metric description info", func(s *mcclient.ClientSession,
		args *MetricUpdateOptions) error {
		updateInput := new(monitorapi.MetricUpdateInput)
		updateInput.Measurement.DisplayName = args.MeasurementDisplayName
		updateInput.Measurement.ResType = args.ResType
		updateField := new(monitorapi.MetricFieldUpdateInput)
		updateField.Name = args.FieldName
		updateField.DisplayName = args.FieldDisplayName
		updateField.Unit = args.Unit
		updateInput.MetricFields = []monitorapi.MetricFieldUpdateInput{*updateField}
		updateInput.Scope = "system"
		result, err := modules.MetricManager.Update(s, args.ID, updateInput.JSON(updateInput))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type MetricShowOptions struct {
		ID string `help:"ID of Metric " json:"-"`
	}
	R(&MetricShowOptions{}, "metric-describ-show", "show metric description info", func(s *mcclient.ClientSession,
		args *MetricShowOptions) error {
		result, err := modules.MetricManager.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
