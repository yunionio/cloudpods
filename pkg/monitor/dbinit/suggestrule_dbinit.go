package dbinit

import (
	monitor "yunion.io/x/onecloud/pkg/apis/monitor"
)

var DiskUnusedCreateInput *monitor.SuggestSysRuleCreateInput
var EipUnusedCreateInput *monitor.SuggestSysRuleCreateInput
var LbUnusedCreateInput *monitor.SuggestSysRuleCreateInput
var OssSecAclCreateInput *monitor.SuggestSysRuleCreateInput
var RdsUnReasonableCreateInput *monitor.SuggestSysRuleCreateInput
var RedisUnReasonableCreateInput *monitor.SuggestSysRuleCreateInput
var OssUnReasonableCreateInput *monitor.SuggestSysRuleCreateInput
var ScaleDownCreateInput *monitor.SuggestSysRuleCreateInput
var SecGroupRuleInCreateInput *monitor.SuggestSysRuleCreateInput
var SnapShotUnusedCreateInput *monitor.SuggestSysRuleCreateInput

var InitRuleCreateInputMap = make(map[string]*monitor.SuggestSysRuleCreateInput)

func init() {
	diskSetting := new(monitor.SSuggestSysAlertSetting)
	diskSetting.DiskUnused = new(monitor.DiskUnused)
	DiskUnusedCreateInput = NewRule("未挂载的云硬盘", "12h", "336h", monitor.DISK_UNUSED, diskSetting)

	eipSetting := new(monitor.SSuggestSysAlertSetting)
	eipSetting.EIPUnused = new(monitor.EIPUnused)
	EipUnusedCreateInput = NewRule("未挂载的弹性公网IP", "12h", "336h", monitor.EIP_UNUSED, eipSetting)

	lbSetting := new(monitor.SSuggestSysAlertSetting)
	lbSetting.LBUnused = new(monitor.LBUnused)
	LbUnusedCreateInput = NewRule("未使用的负载均衡实例", "12h", "336h", monitor.LB_UNUSED, lbSetting)

	OssSecAclCreateInput = NewRule("对象存储权限为开放读、写的存储桶和文件", "12h", "336h", monitor.OSS_SEC_ACL,
		new(monitor.SSuggestSysAlertSetting))

	redisSetting := new(monitor.SSuggestSysAlertSetting)
	scaleRule := monitor.Scale{
		Database:    "telegraf",
		Measurement: "dcs_cachekeys",
		Operator:    "and",
		Field:       "key_count",
		EvalType:    "<",
		Threshold:   100,
	}
	redisSetting.ScaleRule = &monitor.ScaleRule{scaleRule}
	RedisUnReasonableCreateInput = NewRule("空闲的redis", "12h", "336h", monitor.REDIS_UNREASONABLE, redisSetting)

	rdsSetting := new(monitor.SSuggestSysAlertSetting)
	rdsSetting.ScaleRule = &monitor.ScaleRule{monitor.Scale{
		Database:    "telegraf",
		Measurement: "rds_cpu",
		Operator:    "and",
		Field:       "usage_active",
		EvalType:    "<",
		Threshold:   5,
	}}
	RdsUnReasonableCreateInput = NewRule("空闲的rds", "12h", "336h", monitor.RDS_UNREASONABLE, rdsSetting)

	ossSetting := new(monitor.SSuggestSysAlertSetting)
	ossSetting.ScaleRule = &monitor.ScaleRule{monitor.Scale{
		Database:    "telegraf",
		Measurement: "oss_req",
		Operator:    "and",
		Field:       "req_count",
		EvalType:    "<",
		Threshold:   100,
	}}
	OssUnReasonableCreateInput = NewRule("空闲的oss", "12h", "336h", monitor.OSS_UNREASONABLE, ossSetting)

	serversetting := new(monitor.SSuggestSysAlertSetting)
	serversetting.ScaleRule = &monitor.ScaleRule{monitor.Scale{
		Database:    "telegraf",
		Measurement: "vm_cpu",
		Operator:    "and",
		Field:       "usage_active",
		EvalType:    "<",
		Threshold:   5,
	}}
	ScaleDownCreateInput = NewRule("低负载的虚拟机", "12h", "336h", monitor.SCALE_DOWN, serversetting)

	SecGroupRuleInCreateInput = NewRule("安全组规则的in规则为全开放的主机", "12h", "336h",
		monitor.SECGROUPRULEINSERVER_ALLIN, &monitor.SSuggestSysAlertSetting{})

	SnapShotUnusedCreateInput = NewRule("未使用的快照", "12h", "336h", monitor.SNAPSHOT_UNUSED,
		&monitor.SSuggestSysAlertSetting{})
}

func NewRule(name, period, timeFrom string, typ monitor.SuggestDriverType, setting *monitor.SSuggestSysAlertSetting) *monitor.
	SuggestSysRuleCreateInput {
	rule := new(monitor.SuggestSysRuleCreateInput)
	enable := false
	rule.Name = name
	rule.Type = string(typ)
	rule.Period = period
	rule.TimeFrom = timeFrom
	rule.Setting = setting
	rule.Enabled = &enable
	return rule
}
