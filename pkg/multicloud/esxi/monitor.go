package esxi

import (
	"strings"
	"time"

	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

var (
	metricIdTable     = map[int32]types.PerfCounterInfo{}
	metricNameTable   = map[string]types.PerfCounterInfo{}
	metricIdNameTable = map[int32]string{}
	metricSpecsMap    map[string][]string
)

func (client *SESXiClient) GetMonitorData(server jsonutils.JSONObject, metricSpecs map[string][]string, since time.Time,
	until time.Time) (perfEntityMetricList []*types.PerfEntityMetric, IdNameTable map[int32]string, err error) {
	metricSpecsMap = metricSpecs
	name, _ := server.GetString("name")
	virtualMachines, err := client.getVirtualMachines()
	if err != nil {
		return nil, IdNameTable, err
	}
	managedEntity := getManagerEntiry(virtualMachines, name)

	performanceManager := performance.NewManager(client.client.Client)
	perfCounters, err := performanceManager.CounterInfo(client.context)
	if err != nil {
		return nil, IdNameTable, err
	}
	loadMap(perfCounters)

	perfProviderSummary, err := performanceManager.ProviderSummary(client.context, managedEntity.Self)
	if err != nil {
		return nil, IdNameTable, err
	}
	refreshInterval := perfProviderSummary.RefreshRate
	perfMetricList, err := performanceManager.AvailableMetric(client.context, managedEntity.Self, refreshInterval)
	pmiList := buildPerfMetricIds(perfMetricList)

	perfQuerySpec := types.PerfQuerySpec{
		Entity:     managedEntity.Reference(),
		MaxSample:  int32(5),
		MetricId:   pmiList,
		Format:     "normal",
		IntervalId: refreshInterval,
		StartTime:  &since,
		EndTime:    &until,
	}
	perfEntityMetrics, err := performanceManager.Query(client.context, []types.PerfQuerySpec{perfQuerySpec})
	if err != nil {
		return nil, IdNameTable, err
	}
	for _, perfEntityMetricBase := range perfEntityMetrics {
		if perfEntityMetric, ok := perfEntityMetricBase.(*types.PerfEntityMetric); ok {
			perfEntityMetricList = append(perfEntityMetricList, perfEntityMetric)
		}
	}
	return perfEntityMetricList, metricIdNameTable, nil
}

func (cli *SESXiClient) getVirtualMachines() ([]mo.VirtualMachine, error) {
	var virtualMachines []mo.VirtualMachine
	err := cli.scanAllMObjects(VIRTUAL_MACHINE_PROPS, &virtualMachines)
	if err != nil {
		return virtualMachines, errors.Wrap(err, "cli.scanAllMObjects host")
	}
	return virtualMachines, nil
}

func getManagerEntiry(virtualMachines []mo.VirtualMachine, name string) *mo.ManagedEntity {
	for _, virtualMachine := range virtualMachines {
		if virtualMachine.Name == name {
			return virtualMachine.Entity()
		}
	}
	return nil
}

//根据perfCounterInfos装载metricIdTable、metricNameTable、metricIdNameTable
func loadMap(perfCounterInfos []types.PerfCounterInfo) {
	for _, perfCounterInfo := range perfCounterInfos {
		metricId := perfCounterInfo.Key
		metricIdTable[metricId] = perfCounterInfo
		if perfCounterInfo.GroupInfo.GetElementDescription() != nil && perfCounterInfo.GroupInfo.
			GetElementDescription().Key != "" {
			var builder strings.Builder
			builder.WriteString(perfCounterInfo.GroupInfo.GetElementDescription().Key)
			if perfCounterInfo.NameInfo.GetElementDescription() != nil && perfCounterInfo.NameInfo.
				GetElementDescription().Key != "" {
				builder.WriteString("_")
				builder.WriteString(perfCounterInfo.NameInfo.GetElementDescription().Key)
			}
			if len(perfCounterInfo.AssociatedCounterId) == 0 && perfCounterInfo.RollupType == types.
				PerfSummaryTypeAverage {
				metricNameTable[builder.String()] = perfCounterInfo
				metricIdNameTable[metricId] = builder.String()
			}
		}
	}
}

//根据传入的metricSpecsMap的进行过滤，只取相关的指标metricName
func buildPerfMetricIds(pmis []types.PerfMetricId) (pmiList []types.PerfMetricId) {
	for _, perfMetricId := range pmis {
		metricId := perfMetricId.CounterId
		if metricName, ok := metricIdNameTable[metricId]; ok && metricName != "" {
			if _, ok := metricSpecsMap[metricName]; ok {
				pmiList = append(pmiList, perfMetricId)
			}
		}
	}
	return pmiList
}
