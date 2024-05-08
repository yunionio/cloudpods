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

	"golang.org/x/text/language"

	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	schapi "yunion.io/x/onecloud/pkg/apis/scheduledtask"
	"yunion.io/x/onecloud/pkg/i18n"
)

type SEventDisplay struct {
	sEvenWebhookMsg
	ResourceTypeDisplay string
	ActionDisplay       string
	AdvanceDays         int
}

type sEvenWebhookMsg struct {
	ResourceType    string                 `json:"resource_type"`
	Action          string                 `json:"action"`
	ResourceDetails map[string]interface{} `json:"resource_details"`
}

func languageTag(lang string) language.Tag {
	var langStr string
	if lang == api.TEMPLATE_LANG_CN {
		langStr = "zh-CN"
	} else {
		langStr = "en"
	}
	t, _ := language.Parse(langStr)
	return t
}

var action2Topic = make(map[string]string, 0)

func init() {
	action2Topic[string(api.ActionRebuildRoot)] = string(api.ActionUpdate)
	action2Topic[string(api.ActionResetPassword)] = string(api.ActionUpdate)
	action2Topic[string(api.ActionChangeIpaddr)] = string(api.ActionUpdate)
}

var specFieldTrans = map[string]i18n.Table{}

func init() {
	var spI18nTable = i18n.Table{}
	spI18nTable.Set(comapi.TRIGGER_ALARM, i18n.NewTableEntry().EN("alarm").CN("告警"))
	spI18nTable.Set(comapi.TRIGGER_TIMING, i18n.NewTableEntry().EN("timing").CN("定时"))
	spI18nTable.Set(comapi.TRIGGER_CYCLE, i18n.NewTableEntry().EN("cycle").CN("周期"))
	spI18nTable.Set(comapi.ACTION_ADD, i18n.NewTableEntry().EN("add").CN("增加"))
	spI18nTable.Set(comapi.ACTION_REMOVE, i18n.NewTableEntry().EN("remove").CN("减少"))
	spI18nTable.Set(comapi.ACTION_SET, i18n.NewTableEntry().EN("set as").CN("设置为"))
	spI18nTable.Set(comapi.UNIT_ONE, i18n.NewTableEntry().EN("").CN("个"))
	spI18nTable.Set(comapi.UNIT_PERCENT, i18n.NewTableEntry().EN("%").CN("%"))

	var stI18nTable = i18n.Table{}
	stI18nTable.Set(schapi.ST_RESOURCE_SERVER, i18n.NewTableEntry().EN("virtual machine").CN("虚拟机"))
	stI18nTable.Set(schapi.ST_RESOURCE_CLOUDACCOUNT, i18n.NewTableEntry().EN("cloud account").CN("云账号"))
	stI18nTable.Set(schapi.ST_RESOURCE_OPERATION_RESTART, i18n.NewTableEntry().EN("restart").CN("重启"))
	stI18nTable.Set(schapi.ST_RESOURCE_OPERATION_STOP, i18n.NewTableEntry().EN("stop").CN("关机"))
	stI18nTable.Set(schapi.ST_RESOURCE_OPERATION_START, i18n.NewTableEntry().EN("start").CN("开机"))
	stI18nTable.Set(schapi.ST_RESOURCE_OPERATION_SYNC, i18n.NewTableEntry().EN("sync").CN("同步"))

	specFieldTrans[api.TOPIC_RESOURCE_SCALINGPOLICY] = spI18nTable
	specFieldTrans[api.TOPIC_RESOURCE_SCHEDULEDTASK] = stI18nTable
}

var (
	notifyclientI18nTable = i18n.Table{}
)

func setI18nTable(t i18n.Table, elems ...sI18nElme) {
	for i := range elems {
		t.Set(elems[i].k, i18n.NewTableEntry().EN(elems[i].en).CN(elems[i].cn))
	}
}

func getLangSuffix(ctx context.Context) string {
	return notifyclientI18nTable.Lookup(ctx, tempalteLang)
}

const (
	tempalteLang = "lang"
)

type sI18nElme struct {
	k  string
	en string
	cn string
}

func init() {
	setI18nTable(notifyclientI18nTable,
		sI18nElme{
			tempalteLang,
			api.TEMPLATE_LANG_EN,
			api.TEMPLATE_LANG_CN,
		},
		sI18nElme{
			api.TOPIC_RESOURCE_HOST,
			"host",
			"宿主机",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SERVER,
			"virtual machine",
			"虚拟机",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SCALINGGROUP,
			"scaling group",
			"弹性伸缩组",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SCALINGPOLICY,
			"scaling policy",
			"弹性伸缩策略",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_IMAGE,
			"image",
			"系统镜像",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_DISK,
			"disk",
			"硬盘",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SNAPSHOT,
			"snapshot",
			"硬盘快照",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_INSTANCESNAPSHOT,
			"instance snapshot",
			"主机快照",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_NETWORK,
			"network",
			"IP子网",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_EIP,
			"EIP",
			"弹性公网IP",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SECGROUP,
			"security group",
			"安全组",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_LOADBALANCER,
			"loadbalancer instance",
			"负载均衡实例",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_LOADBALANCERACL,
			"loadbalancer ACL",
			"负载均衡访问控制",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE,
			"loadbalancer certificate",
			"负载均衡证书",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_LOADBALANCERLISTENER,
			"loadbalancer listener",
			"负载均衡监听",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_LOADBALANCERBACKEDNGROUP,
			"loadbalancer backendgroup",
			"负载均衡服务器组",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_BUCKET,
			"object storage bucket",
			"对象存储桶",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_DBINSTANCE,
			"RDS instance",
			"RDS实例",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_ELASTICCACHE,
			"Redis instance",
			"Redis实例",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SCHEDULEDTASK,
			"scheduled task",
			"定时任务",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_BAREMETAL,
			"baremetal",
			"裸金属",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SNAPSHOTPOLICY,
			"snapshot policy",
			"快照策略",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_VPC,
			"VPC",
			"VPC",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_DNSZONE,
			"DNS zone",
			"DNS zone",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_DNSRECORDSET,
			"DNS record",
			"DNS 记录",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_NATGATEWAY,
			"nat gateway",
			"nat网关",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_WEBAPP,
			"webapp",
			"应用程序服务",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_CDNDOMAIN,
			"CDN domain",
			"CDN domain",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_FILESYSTEM,
			"file system",
			"文件系统",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_WAF,
			"WAF",
			"WAF",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_KAFKA,
			"Kafka",
			"Kafka",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_ELASTICSEARCH,
			"Elasticsearch",
			"Elasticsearch",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_MONGODB,
			"MongoDB",
			"MongoDB",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_DB_TABLE_RECORD,
			"database table record",
			"数据库记录",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_VM_INTEGRITY_CHECK,
			"vm server integrity check",
			"虚拟主机完整性校验",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_ACTION_LOG,
			"action log",
			"操作日志",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_PROJECT,
			"project",
			"项目",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_CLOUDPODS_COMPONENT,
			"cloudpods component",
			"cloudpods服务组件",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_USER,
			"user",
			"用户",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_CLOUDPHONE,
			"cloudphone",
			"云手机",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_ACCOUNT_STATUS,
			"account",
			"云账号",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SERVICE,
			"service",
			"服务",
		},
		sI18nElme{
			string(api.ActionCreate),
			"created",
			"创建",
		},
		sI18nElme{
			string(api.ActionUpdate),
			"update",
			"更新",
		},
		sI18nElme{
			string(api.ActionDelete),
			"deleted",
			"删除",
		},
		sI18nElme{
			string(api.ActionRebuildRoot),
			"rebuilded root",
			"重装系统",
		},
		sI18nElme{
			string(api.ActionResetPassword),
			"reseted password",
			"重置密码",
		},
		sI18nElme{
			string(api.ActionChangeConfig),
			"changed config",
			"更改配置",
		},
		sI18nElme{
			string(api.ActionResize),
			"resize",
			"扩容",
		},
		sI18nElme{
			string(api.ActionExpiredRelease),
			"expired and released",
			"到期释放",
		},
		sI18nElme{
			string(api.ActionExecute),
			"executed",
			"生效执行",
		},
		sI18nElme{
			string(api.ActionPendingDelete),
			"added to the recycle bin",
			"加入回收站",
		},
		sI18nElme{
			string(api.ActionSyncCreate),
			"sync_create",
			"同步新建",
		},
		sI18nElme{
			string(api.ActionSyncUpdate),
			"sync_update",
			"同步更新",
		},
		sI18nElme{
			string(api.ActionSyncDelete),
			"sync_delete",
			"同步删除",
		},
		sI18nElme{
			string(api.ActionMigrate),
			"migrate",
			"迁移",
		},
		sI18nElme{
			string(api.ActionOffline),
			"offline",
			"离线",
		},
		sI18nElme{
			string(api.ActionSystemException),
			"exception",
			"异常",
		},
		sI18nElme{
			string(api.ResultFailed),
			"failed",
			"失败",
		},
		sI18nElme{
			string(api.ResultSucceed),
			"successfully",
			"成功",
		},
		sI18nElme{
			string(api.ActionAttach),
			"attach",
			"挂载",
		},
		sI18nElme{
			string(api.ActionDetach),
			"detach",
			"卸载",
		},
		sI18nElme{
			string(api.ActionCreateBackupServer),
			"add_backup_server",
			"添加主机备份",
		},
		sI18nElme{
			string(api.ActionStart),
			"start",
			"开机",
		},
		sI18nElme{
			string(api.ActionStop),
			"stop",
			"关机",
		},
		sI18nElme{
			string(api.ActionRestart),
			"restart",
			"重启",
		},
		sI18nElme{
			string(api.ActionReset),
			"reset",
			"重置",
		},
		sI18nElme{
			string(api.ActionChangeIpaddr),
			"change_ipaddr",
			"修改IP地址",
		},
		sI18nElme{
			string(api.ActionChecksumTest),
			"checksum_test",
			"一致性检查",
		},
		sI18nElme{
			string(api.ActionCleanData),
			"clean_data",
			"清理数据",
		},
		sI18nElme{
			string(api.ActionDelBackupServer),
			"delete_backup_server",
			"删除主机备份",
		},
		sI18nElme{
			string(api.ActionMysqlOutOfSync),
			"mysql_out_of_sync",
			"数据库不一致",
		},
		sI18nElme{
			string(api.ActionNetOutOfSync),
			"net_out_of_sync",
			"网络拓扑不一致",
		},
		sI18nElme{
			string(api.ActionServerPanicked),
			"server_panicked",
			"主机崩溃",
		},
		sI18nElme{
			string(api.ActionServiceAbnormal),
			"service_abnormal",
			"服务异常",
		},
		sI18nElme{
			string(api.ActionPasswordExpireSoon),
			"password_expire_soon",
			"密码即将过期",
		},
		sI18nElme{
			string(api.ActionSystemPanic),
			"panic",
			"系统崩溃",
		},
		sI18nElme{
			string(api.ActionExceedCount),
			"exceed_count",
			"超过数量",
		},
		sI18nElme{
			string(api.ActionIsolatedDeviceCreate),
			"isolated_device_create",
			"新增透传设备",
		},
		sI18nElme{
			string(api.ActionIsolatedDeviceUpdate),
			"isolated_device_update",
			"修改透传设备",
		},
		sI18nElme{
			string(api.ActionIsolatedDeviceDelete),
			"isolated_device_delete",
			"删除透传设备",
		},

		sI18nElme{
			string(api.ActionStatusChanged),
			"status_changed",
			"状态变更",
		},
	)
}
