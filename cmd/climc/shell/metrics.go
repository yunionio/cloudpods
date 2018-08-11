package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 列出指定指标类型下的全部指标
	 */
	type MetricTypesBaseOptions struct {
		ID string `help:"ID of the metric type"`
	}
	R(&MetricTypesBaseOptions{}, "metrictype-metric-list", "List metric types of the monitor type", func(s *mcclient.ClientSession, args *MetricTypesBaseOptions) error {
		result, err := modules.Metrics.ListInContext(s, nil, &modules.MetricsTypes, args.ID)
		if err != nil {
			return err
		}
		printList(result, modules.Metrics.GetColumns(s))
		return nil
	})

	/**
	 * 列出所有监控指标
	 */
	type MetricsListOptions struct {
		BaseListOptions
	}
	R(&MetricsListOptions{}, "metric-list", "List all metrics", func(s *mcclient.ClientSession, args *MetricsListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.Metrics.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.Metrics.GetColumns(s))
		return nil
	})

	/**
	 * 查看监控指标详情
	 */
	type MetricsShowOptions struct {
		BaseListOptions
		ID string `help:"The ID of the metric"`
	}
	R(&MetricsShowOptions{}, "metric-show", "Show metric details", func(s *mcclient.ClientSession, args *MetricsShowOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.Metrics.Get(s, args.ID, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	/**
	 * 根据name查看监控指标详情
	 */
	type MetricsShowByNameOptions struct {
		BaseListOptions
		NAME string `help:"The NAME of the metric"`
	}
	R(&MetricsShowByNameOptions{}, "metric-details", "Show metric details by name", func(s *mcclient.ClientSession, args *MetricsShowByNameOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.Metrics.GetSpecific(s, "", args.NAME, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

}
