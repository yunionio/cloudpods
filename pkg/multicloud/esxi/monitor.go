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

package esxi

import (
	"fmt"
	"strings"
	"time"

	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SPerfMetricInfo struct {
	metricIdTable     map[int32]types.PerfCounterInfo
	metricNameTable   map[string]types.PerfCounterInfo
	metricIdNameTable map[int32]string
	metricSpecsMap    map[string][]string
}

func (client *SESXiClient) GetMonitorData(serverOrHost jsonutils.JSONObject,
	monType string, metricSpecs map[string][]string,
	since time.Time, until time.Time) (perfEntityMetricList []*types.PerfEntityMetric, IdNameTable map[int32]string, err error) {
	perMetricInfo := SPerfMetricInfo{
		metricIdTable:     make(map[int32]types.PerfCounterInfo),
		metricNameTable:   make(map[string]types.PerfCounterInfo),
		metricIdNameTable: make(map[int32]string),
		metricSpecsMap:    make(map[string][]string),
	}
	perMetricInfo.metricSpecsMap = metricSpecs
	managedEntity, err := client.getManagerEntiry(serverOrHost, monType)
	if err != nil {
		return nil, IdNameTable, err
	}

	performanceManager := performance.NewManager(client.client.Client)
	perfCounters, err := performanceManager.CounterInfo(client.context)
	if err != nil {
		return nil, IdNameTable, err
	}
	perMetricInfo = loadMap(perfCounters, perMetricInfo)
	//log.Errorf("metricIdNameInfo:%v", metricIdNameTable)
	perfProviderSummary, err := performanceManager.ProviderSummary(client.context, managedEntity.Self)
	if err != nil {
		return nil, IdNameTable, err
	}
	refreshInterval := perfProviderSummary.RefreshRate
	perfMetricList, err := performanceManager.AvailableMetric(client.context, managedEntity.Self, refreshInterval)
	//log.Errorf("availableMetric:%v", perfMetricList)
	pmiList := buildPerfMetricIds(perfMetricList, perMetricInfo)
	//log.Errorf("filteredMetric:%v", pmiList)

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
	return perfEntityMetricList, perMetricInfo.metricIdNameTable, nil
}

func (client *SESXiClient) GetMonitorDataList(serverOrHost []jsonutils.JSONObject,
	monType string, metrics []string,
	since time.Time, until time.Time) (map[string]performance.EntityMetric, error) {
	entityServerorm := make(map[string]string)
	entityRefs := make([]types.ManagedObjectReference, 0)
	for i, _ := range serverOrHost {
		managedEntity, err := client.getManagerEntiry(serverOrHost[i], monType)
		if err != nil {
			log.Errorln(errors.Wrap(err, "getManagerEntiry error"))
			continue
		}
		extId, _ := serverOrHost[i].GetString("external_id")
		entityServerorm[managedEntity.Self.Value] = extId
		entityRefs = append(entityRefs, managedEntity.Self)
	}

	performanceManager := performance.NewManager(client.client.Client)
	counterInfoNameMap, err := performanceManager.CounterInfoByName(client.context)
	names := make([]string, 0)
	ids := make(map[int32]string)
	for _, metric := range metrics {
		if couterInfo, ok := counterInfoNameMap[metric]; ok {
			names = append(names, metric)
			ids[couterInfo.Key] = ""
		}
	}

	if err != nil {
		return nil, err
	}
	perfProviderSummary, err := performanceManager.ProviderSummary(client.context, entityRefs[0])
	if err != nil {
		return nil, err
	}
	refreshInterval := perfProviderSummary.RefreshRate
	perfMetricList, err := performanceManager.AvailableMetric(client.context, entityRefs[0], refreshInterval)
	pmis := getPerfMetrics(perfMetricList, ids)
	perfQuerySpec := types.PerfQuerySpec{
		MaxSample:  int32(5),
		MetricId:   pmis,
		Format:     "normal",
		IntervalId: refreshInterval,
		StartTime:  &since,
		EndTime:    &until,
	}
	metricEntities, err := performanceManager.SampleByName(client.context, perfQuerySpec, names, entityRefs)
	if err != nil {
		return nil, err
	}
	result, err := performanceManager.ToMetricSeries(client.context, metricEntities)
	if err != nil {
		return nil, err
	}
	resultMap := make(map[string]performance.EntityMetric)
	for i, res := range result {
		if extId, ok := entityServerorm[res.Entity.Value]; ok {
			resultMap[extId] = result[i]
		}
	}
	return resultMap, nil
}

func (cli *SESXiClient) getVirtualMachines() ([]mo.VirtualMachine, error) {
	var virtualMachines []mo.VirtualMachine
	err := cli.scanAllMObjects(VIRTUAL_MACHINE_PROPS, &virtualMachines)
	if err != nil {
		return virtualMachines, errors.Wrap(err, "cli.scanAllMObjects host")
	}
	return virtualMachines, nil
}

func (cli *SESXiClient) GetHostSystem() ([]mo.HostSystem, error) {
	var hostSystems []mo.HostSystem
	err := cli.scanAllMObjects(HOST_SYSTEM_PROPS, &hostSystems)
	if err != nil {
		return hostSystems, errors.Wrap(err, "cli.scanAllMObjects host")
	}
	return hostSystems, nil
}

func (cli *SESXiClient) getManagerEntiry(serverOrHost jsonutils.JSONObject, monType string) (*mo.ManagedEntity, error) {
	switch monType {
	case "VirtualMachine":
		return cli.getManagerEntityofVm(serverOrHost)
	case "HostSystem":
		return cli.getManagerEntityofHost(serverOrHost)
	}
	return nil, fmt.Errorf("monType error")
}

func (cli *SESXiClient) getManagerEntityofVm(server jsonutils.JSONObject) (*mo.ManagedEntity, error) {
	name, _ := server.GetString("name")
	extId, _ := server.GetString("external_id")
	virtualMachines, err := cli.getVirtualMachines()
	if err != nil {
		return nil, err
	}
	for _, virtualMachine := range virtualMachines {
		guest := NewVirtualMachine(cli, &virtualMachine, nil)
		ip := guest.GetGlobalId()
		if ip == extId {
			return virtualMachine.Entity(), nil
		}
	}
	return nil, fmt.Errorf("No ManagerEntiry for %s vm", name)
}

func (cli *SESXiClient) getManagerEntityofHost(host jsonutils.JSONObject) (*mo.ManagedEntity, error) {
	name, _ := host.GetString("name")
	extId, _ := host.GetString("external_id")
	hostSystems, err := cli.GetHostSystem()
	if err != nil {
		return nil, err
	}
	for _, hostSystem := range hostSystems {
		host := NewHost(cli, &hostSystem, nil)
		ip := host.GetGlobalId()
		if ip == extId {
			return hostSystem.Entity(), nil
		}
	}
	return nil, fmt.Errorf("No VMware ManagerEntiry for %s host", name)
}

//根据perfCounterInfos装载metricIdTable、metricNameTable、metricIdNameTable
func loadMap(perfCounterInfos []types.PerfCounterInfo, perfMetricInfo SPerfMetricInfo) SPerfMetricInfo {

	for _, perfCounterInfo := range perfCounterInfos {
		metricId := perfCounterInfo.Key
		perfMetricInfo.metricIdTable[metricId] = perfCounterInfo
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
				perfMetricInfo.metricNameTable[builder.String()] = perfCounterInfo
				perfMetricInfo.metricIdNameTable[metricId] = builder.String()
			}
		}
	}
	return perfMetricInfo
}

//根据传入的metricSpecsMap的进行过滤，只取相关的指标metricName
func buildPerfMetricIds(pmis []types.PerfMetricId, perfMetricInfo SPerfMetricInfo) (pmiList []types.PerfMetricId) {
	for i, perfMetricId := range pmis {
		metricId := perfMetricId.CounterId
		if metricName, ok := perfMetricInfo.metricIdNameTable[metricId]; ok && metricName != "" {
			if _, ok := perfMetricInfo.metricSpecsMap[metricName]; ok {
				pmiList = append(pmiList, pmis[i])
			}
		}
	}
	return pmiList
}

func getPerfMetrics(pmis []types.PerfMetricId, ids map[int32]string) (pmiList []types.PerfMetricId) {
	for i, pmi := range pmis {
		if _, ok := ids[pmi.CounterId]; ok {
			pmiList = append(pmiList, pmis[i])
		}
	}
	return
}
