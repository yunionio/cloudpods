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

package measurements

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

var All = []SMeasurement{
	bond,
	bondSlave,

	cpu,
	disk,
	diskio,
	linuxSysctlFs,
	mem,
	net,
	netint,
	nvidia,
	processes,
	radeontop,
	system,
	vasmi,
	npuSmi,
	hysmi,
	ixsmi,
	ppusmi,
	xpusmi,

	worker,
	serviceHttpCode,
	serviceProcessStats,
	dbStats,
	statusProbe,

	vmCpu,
	vmMem,
	vmDisk,
	vmDiskio,
	vmNetio,

	cloudAccount,

	containerCpu,
	containerMem,
	containerProcess,
	containerDiskIo,

	dcsRedisCpu,
	dcsRedisMem,
	dcsRedisNetio,
	dcsRedisConn,
	dcsRedisInstanceOpt,
	dcsRedisCachekeys,
	dcsRedisDatamem,

	haproxy,

	internalAgent,
	internalGather,
	internalMemstats,
	internalWrite,

	jenkinsNode,
	jenkinsJob,

	k8sPod,
	k8sDeploy,
	k8sNode,

	kernel,
	kernelVmstat,

	mysql,

	netstat,

	ntpq,

	ossLatency,
	ossNetio,
	ossReq,
	ossPerfMon,

	ping,

	podCpu,
	podGpu,
	podMem,
	podVolume,
	podProcess,
	podDiskIo,

	rabbitmqOverview,
	rabbitmqNode,
	rabbitmqQueue,

	rdsConn,
	rdsCpu,
	rdsMem,
	rdsNetio,
	rdsDisk,

	redis,
	redisKeyspace,

	sensors,
	ipmiSensor,

	smartctl,

	storage,

	swap,

	temp,

	hostRaid,
}

func newHostAgentMonitorContext(name string, displayName string) []SMonitorContext {
	return []SMonitorContext{
		{
			Name:         name,
			DisplayName:  displayName,
			ResourceType: monitor.METRIC_RES_TYPE_HOST,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
		{
			Name:         fmt.Sprintf("agent_%s", name),
			DisplayName:  displayName,
			ResourceType: monitor.METRIC_RES_TYPE_AGENT,
			Database:     monitor.METRIC_DATABASE_TELE,
		},
	}
}
