package consts

import (
	"yunion.io/x/onecloud/pkg/i18n"
)

var (
	ResourceTranslateI18nTable = i18n.Table{}
)

func init() {
	resourceTranslate := map[string]map[string]string{
		"DBAudit": {
			"cn": "数据库审计DBAudit",
			"en": "DBAudit",
		},
		"FortressAircraft": {
			"cn": "堡垒机",
			"en": "FortressAircraft",
		},
		"WTPS": {
			"cn": "网页防篡改服务WTPS",
			"en": "Anti-tampering service for web pages",
		},
		"acr": {
			"cn": "容器镜像服务",
			"en": "Acr",
		},
		"activate": {
			"cn": "Activate",
			"en": "Activate",
		},
		"ads": {
			"cn": "分析型数据库",
			"en": "ADS",
		},
		"alikafka": {
			"cn": "消息队列Kafka版",
			"en": "Alikafka",
		},
		"alimail": {
			"cn": "云邮箱",
			"en": "Alimail",
		},
		"anti ddos": {
			"cn": "Anti-DDoS流量清洗",
			"en": "Anti DDoS",
		},
		"aos": {
			"cn": "编排服务",
			"en": "AOS",
		},
		"app service": {
			"cn": "应用服务",
			"en": "App Service",
		},
		"application gateway": {
			"cn": "应用网关",
			"en": "Application Gateway",
		},
		"arms": {
			"cn": "应用实时监控服务",
			"en": "Application real-time monitoring service",
		},
		"automation": {
			"cn": "自动化",
			"en": "Automation",
		},
		"backup": {
			"cn": "备份",
			"en": "Backup",
		},
		"baremetal": {
			"cn": "裸金属服务器",
			"en": "Baremetal",
		},
		"bastionhost": {
			"cn": "运维安全中心（堡垒机）",
			"en": "Operation and maintenance security center (fortress machine)",
		},
		"bi": {
			"cn": "数据可视化",
			"en": "BI",
		},
		"bigquery": {
			"cn": "BigQuery",
			"en": "BigQuery",
		},
		"cache": {
			"cn": "缓存服务",
			"en": "Cache",
		},
		"cas": {
			"cn": "云盾证书服务",
			"en": "CAS",
		},
		"cbs": {
			"cn": "数据库备份DBS",
			"en": "Cbs",
		},
		"cbwp": {
			"cn": "共享带宽",
			"en": "Cbwq",
		},
		"cdn": {
			"cn": "内容分发网络",
			"en": "CDN",
		},
		"clickhouse": {
			"cn": "云数据库ClickHouse",
			"en": "ClickHouse",
		},
		"cloud connect network": {
			"cn": "云联网",
			"en": "Cloud Connect Network",
		},
		"cloud infinite": {
			"cn": "数据万象",
			"en": "Cloud Infinite",
		},
		"cloud streaming services": {
			"cn": "云直播",
			"en": "Cloud Streaming Services",
		},
		"cloud visualization": {
			"cn": "腾讯云图",
			"en": "Cloud Visualization",
		},
		"cloudkms": {
			"cn": "Cloud KMS",
			"en": "Cloud KMS",
		},
		"cloudtrace": {
			"cn": "云审计",
			"en": "CloudTrace",
		},
		"cloudtrail": {
			"cn": "CloudTrail",
			"en": "CloudTrail",
		},
		"cloudwatch": {
			"cn": "CloudWatch",
			"en": "CloudWatch",
		},
		"cms": {
			"cn": "云监控",
			"en": "Cms",
		},
		"config": {
			"cn": "Config",
			"en": "Config",
		},
		"container": {
			"cn": "容器镜像",
			"en": "Container",
		},
		"cross region connection": {
			"cn": "跨地域互联",
			"en": "Cross Region Connection",
		},
		"csk": {
			"cn": "容器服务ACK",
			"en": "ACK",
		},
		"data factory": {
			"cn": "数据工厂",
			"en": "Data Factory",
		},
		"database": {
			"cn": "数据库",
			"en": "Database",
		},
		"datahub": {
			"cn": "数据总线 DataHub",
			"en": "DataHub",
		},
		"datav": {
			"cn": "DataV数据可视化",
			"en": "DataV",
		},
		"dcdn": {
			"cn": "全站加速",
			"en": "DCDN",
		},
		"des": {
			"cn": "数据加密服务",
			"en": "DES",
		},
		"dide": {
			"cn": "大数据开发治理平台 DataWorks",
			"en": "Big data development and governance platform DataWorks",
		},
		"directconnect": {
			"cn": "专线接入",
			"en": "DirectConnect",
		},
		"disk": {
			"cn": "块存储",
			"en": "Block storage",
		},
		"dms": {
			"cn": "数据管理",
			"en": "Data management",
		},
		"dns": {
			"cn": "域名解析服务",
			"en": "DNS",
		},
		"domain": {
			"cn": "域名",
			"en": "Domain",
		},
		"dts": {
			"cn": "数据传输",
			"en": "DTS",
		},
		"dysms": {
			"cn": "短信服务",
			"en": "SMS service",
		},
		"eci": {
			"cn": "弹性容器实例 ECI",
			"en": "ECI",
		},
		"ecs": {
			"cn": "云服务器 ECS",
			"en": "ECS",
		},
		"edge computing machine": {
			"cn": "边缘计算机器",
			"en": "Edge Computing Machine",
		},
		"eip": {
			"cn": "Eip带宽",
			"en": "EIP",
		},
		"elasticsearch": {
			"cn": "检索分析服务 Elasticsearch版",
			"en": "Search Analysis Service Elasticsearch Edition",
		},
		"emapreduce": {
			"cn": "开源大数据平台 E-MapReduce",
			"en": "Open source big data platform E-MapReduce",
		},
		"english composition correction": {
			"cn": "英文作文批改",
			"en": "English Composition Correction",
		},
		"expressconnect": {
			"cn": "高速通道",
			"en": "High speed channel",
		},
		"face recognition": {
			"cn": "人脸识别",
			"en": "Face Recognition",
		},
		"fc": {
			"cn": "函数计算",
			"en": "Fc",
		},
		"file storage": {
			"cn": "文件存储",
			"en": "File Storage",
		},
		"flowbag": {
			"cn": "共享流量包",
			"en": "Shared traffic package",
		},
		"gaap": {
			"cn": "全球应用加速",
			"en": "GAAP",
		},
		"gallery": {
			"cn": "共享映像库",
			"en": "Gallery",
		},
		"gws": {
			"cn": "无影云桌面",
			"en": "Shadowless Cloud Desktop",
		},
		"hbr": {
			"cn": "混合云备份服务",
			"en": "Hybrid cloud backup service",
		},
		"hbrpost": {
			"cn": "混合云备份",
			"en": "Hbrpost",
		},
		"hdm": {
			"cn": "数据库自治服务",
			"en": "Database Autonomy Service",
		},
		"hitsdb": {
			"cn": "时序数据库 InfluxDB® 版",
			"en": "Hitsdb",
		},
		"hologram": {
			"cn": "实时数仓Hologres",
			"en": "Hologres",
		},
		"host": {
			"cn": "专用宿主机",
			"en": "Host",
		},
		"idaas": {
			"cn": "应用身份服务",
			"en": "Idaas",
		},
		"image": {
			"cn": "镜像服务",
			"en": "Image",
		},
		"imm": {
			"cn": "智能媒体管理",
			"en": "Intelligent Media Management",
		},
		"instant messaging": {
			"cn": "即时通信",
			"en": "Instant Messaging",
		},
		"ipv6gateway": {
			"cn": "IPv6 网关",
			"en": "IPv6 Gateway",
		},
		"kafka": {
			"cn": "消息服务Kafka",
			"en": "Kafka",
		},
		"keymanager": {
			"cn": "Key-Management-Service",
			"en": "KeyManager",
		},
		"kms": {
			"cn": "密钥管理服务",
			"en": "Key management service",
		},
		"kvstore": {
			"cn": "云数据库 Redis 版",
			"en": "ApsaraDB for Redis",
		},
		"lambda": {
			"cn": "AWS-Lambda",
			"en": "Lambda",
		},
		"lb": {
			"cn": "负载均衡",
			"en": "LB",
		},
		"live": {
			"cn": "视频直播",
			"en": "Live video",
		},
		"live video broadcasting": {
			"cn": "移动直播连麦",
			"en": "Live Video Broadcasting",
		},
		"log": {
			"cn": "日志服务",
			"en": "Log",
		},
		"lvwang": {
			"cn": "内容安全",
			"en": "Content security",
		},
		"machine learning": {
			"cn": "机器学习",
			"en": "Machine Learning",
		},
		"mapreduce": {
			"cn": "MapReduce",
			"en": "MapReduce",
		},
		"market": {
			"cn": "云市场",
			"en": "Market",
		},
		"mem": {
			"cn": "虚拟机内存",
			"en": "Vminstance Memory",
		},
		"mongodb": {
			"cn": "云数据库MongoDB",
			"en": "MongoDB",
		},
		"monitor": {
			"cn": "监控",
			"en": "Monitor",
		},
		"mqs": {
			"cn": "消息队列服务",
			"en": "MQS",
		},
		"mse": {
			"cn": "微服务引擎 MSE ",
			"en": "MSE ",
		},
		"msg": {
			"cn": "消息通知服务",
			"en": "MSG",
		},
		"nas": {
			"cn": "文件存储NAS",
			"en": "File Storage NAS",
		},
		"nat": {
			"cn": "NAT网关",
			"en": "NAT",
		},
		"notification": {
			"cn": "Notification",
			"en": "Notification",
		},
		"ntr": {
			"cn": "转发路由器",
			"en": "NTR",
		},
		"ocr": {
			"cn": "文字识别",
			"en": "OCR",
		},
		"odps": {
			"cn": "云原生大数据计算服务 MaxCompute",
			"en": "MaxCompute",
		},
		"ons": {
			"cn": "消息队列 RabbitMQ 版",
			"en": "Ons",
		},
		"oss": {
			"cn": "对象存储",
			"en": "OSS",
		},
		"other": {
			"cn": "其他服务",
			"en": "Other Services",
		},
		"polardb": {
			"cn": "云原生关系型数据库 PolarDB",
			"en": "PolarDB",
		},
		"premiumsupport": {
			"cn": "Premium-Support",
			"en": "PremiumSupport",
		},
		"prometheus": {
			"cn": "Prometheus监控服务",
			"en": "Prometheus",
		},
		"pvtz": {
			"cn": "内网DNS解析",
			"en": "DNS Private Zone",
		},
		"quickbi": {
			"cn": "敏捷商业智能报表",
			"en": "Quickbi",
		},
		"rds": {
			"cn": "关系型数据库",
			"en": "Relational Database",
		},
		"rds_disk": {
			"cn": "RDS磁盘",
			"en": "RDS Disk",
		},
		"real time communication": {
			"cn": "实时音视频",
			"en": "Real Time Communication",
		},
		"redshift": {
			"cn": "Redshift",
			"en": "Redshift",
		},
		"reseller": {
			"cn": "转销",
			"en": "Reseller",
		},
		"ri": {
			"cn": "预留实例",
			"en": "RI",
		},
		"rounding": {
			"cn": "计费精度差异",
			"en": "Rounding",
		},
		"saf": {
			"cn": "风险识别",
			"en": "Risk identification",
		},
		"sc": {
			"cn": "flink全托管",
			"en": "SC",
		},
		"scalinggroup": {
			"cn": "弹性伸缩组",
			"en": "ScalingGroup",
		},
		"security center": {
			"cn": "安全中心",
			"en": "Security Center",
		},
		"server": {
			"cn": "虚拟机",
			"en": "Server",
		},
		"serversecurity": {
			"cn": "主机安全",
			"en": "ServerSecurity",
		},
		"servicestage": {
			"cn": "应用管理与运维平台",
			"en": "ServiceStage",
		},
		"sfs": {
			"cn": "弹性文件服务",
			"en": "SFS",
		},
		"slb": {
			"cn": "负载均衡",
			"en": "Load balancing",
		},
		"sls": {
			"cn": "日志服务",
			"en": "Sls",
		},
		"smart oral evaluation": {
			"cn": "口语评测",
			"en": "Smart Oral Evaluation",
		},
		"smartag": {
			"cn": "智能接入网关APP",
			"en": "Smartag",
		},
		"sms": {
			"cn": "短信",
			"en": "SMS",
		},
		"snapshot": {
			"cn": "快照",
			"en": "Snapshot",
		},
		"sourcerepo": {
			"cn": "SourceRepo",
			"en": "SourceRepo",
		},
		"speech recognition": {
			"cn": "语音识别",
			"en": "Speech Recognition",
		},
		"ssl certificate": {
			"cn": "SSL证书",
			"en": "SSL Certificate",
		},
		"storage": {
			"cn": "存储",
			"en": "Storage",
		},
		"storage account": {
			"cn": "azure对象存储",
			"en": "Storage Account",
		},
		"support": {
			"cn": "Support",
			"en": "Support",
		},
		"t-sec": {
			"cn": "T-Sec",
			"en": "T-Sec",
		},
		"t-sec anti ddos": {
			"cn": "T-Sec DDoS",
			"en": "T-Sec Anti DDoS",
		},
		"t-sec cwp": {
			"cn": "T-Sec主机安全",
			"en": "T-Sec CWP",
		},
		"tax": {
			"cn": "税金",
			"en": "Tax",
		},
		"translate": {
			"cn": "翻译服务",
			"en": "Translate",
		},
		"tsdb": {
			"cn": "时序数据库",
			"en": "TSDB",
		},
		"vault": {
			"cn": "保管库",
			"en": "Vault",
		},
		"video on demand": {
			"cn": "点播",
			"en": "Video On Demand",
		},
		"vod": {
			"cn": "视频点播",
			"en": "Video on demand",
		},
		"voice message": {
			"cn": "语音消息",
			"en": "Voice Message",
		},
		"vpc": {
			"cn": "VPC",
			"en": "VPC",
		},
		"vpn": {
			"cn": "VPN网关",
			"en": "VPN",
		},
		"vpn gateway": {
			"cn": "VPN网关",
			"en": "VPN Gateway",
		},
		"waf": {
			"cn": "Web应用防火墙",
			"en": "Web application firewall",
		},
		"workflow": {
			"cn": "工作流",
			"en": "Workflow",
		},
		"xtrace": {
			"cn": "链路追踪",
			"en": "Link Tracking",
		},
	}
	for resource, translate := range resourceTranslate {
		ResourceTranslateI18nTable.Set(resource, i18n.NewTableEntry().CN(translate["cn"]).EN(translate["en"]))
	}
}
