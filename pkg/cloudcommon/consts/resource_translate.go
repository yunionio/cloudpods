package consts

import (
	"yunion.io/x/onecloud/pkg/i18n"
)

var (
	ResourceTranslateI18nTable = i18n.Table{}
)

func init() {
	type slang struct {
		CN string
		EN string
	}
	resourceTranslate := map[string]slang{
		"DBAudit": {
			CN: "数据库审计DBAudit",
			EN: "DBAudit",
		},
		"FortressAircraft": {
			CN: "堡垒机",
			EN: "FortressAircraft",
		},
		"WTPS": {
			CN: "网页防篡改服务WTPS",
			EN: "Anti-tampering service for web pages",
		},
		"acr": {
			CN: "容器镜像服务",
			EN: "Acr",
		},
		"activate": {
			CN: "Activate",
			EN: "Activate",
		},
		"ads": {
			CN: "分析型数据库",
			EN: "ADS",
		},
		"alikafka": {
			CN: "消息队列Kafka版",
			EN: "Alikafka",
		},
		"alimail": {
			CN: "云邮箱",
			EN: "Alimail",
		},
		"anti ddos": {
			CN: "Anti-DDoS流量清洗",
			EN: "Anti DDoS",
		},
		"aos": {
			CN: "编排服务",
			EN: "AOS",
		},
		"app service": {
			CN: "应用服务",
			EN: "App Service",
		},
		"application gateway": {
			CN: "应用网关",
			EN: "Application Gateway",
		},
		"arms": {
			CN: "应用实时监控服务",
			EN: "Application real-time monitoring service",
		},
		"automation": {
			CN: "自动化",
			EN: "Automation",
		},
		"backup": {
			CN: "备份",
			EN: "Backup",
		},
		"baremetal": {
			CN: "裸金属服务器",
			EN: "Baremetal",
		},
		"bastionhost": {
			CN: "运维安全中心（堡垒机）",
			EN: "Operation and maintenance security center (fortress machine)",
		},
		"bi": {
			CN: "数据可视化",
			EN: "BI",
		},
		"bigquery": {
			CN: "BigQuery",
			EN: "BigQuery",
		},
		"cache": {
			CN: "缓存服务",
			EN: "Cache",
		},
		"cas": {
			CN: "云盾证书服务",
			EN: "CAS",
		},
		"cbs": {
			CN: "数据库备份DBS",
			EN: "Cbs",
		},
		"cbwp": {
			CN: "共享带宽",
			EN: "Cbwq",
		},
		"cdn": {
			CN: "内容分发网络",
			EN: "CDN",
		},
		"clickhouse": {
			CN: "云数据库ClickHouse",
			EN: "ClickHouse",
		},
		"cloud connect network": {
			CN: "云联网",
			EN: "Cloud Connect Network",
		},
		"cloud infinite": {
			CN: "数据万象",
			EN: "Cloud Infinite",
		},
		"cloud streaming services": {
			CN: "云直播",
			EN: "Cloud Streaming Services",
		},
		"cloud visualization": {
			CN: "腾讯云图",
			EN: "Cloud Visualization",
		},
		"cloudkms": {
			CN: "Cloud KMS",
			EN: "Cloud KMS",
		},
		"cloudtrace": {
			CN: "云审计",
			EN: "CloudTrace",
		},
		"cloudtrail": {
			CN: "CloudTrail",
			EN: "CloudTrail",
		},
		"cloudwatch": {
			CN: "CloudWatch",
			EN: "CloudWatch",
		},
		"cms": {
			CN: "云监控",
			EN: "Cms",
		},
		"config": {
			CN: "Config",
			EN: "Config",
		},
		"container": {
			CN: "容器镜像",
			EN: "Container",
		},
		"cross region connection": {
			CN: "跨地域互联",
			EN: "Cross Region Connection",
		},
		"csk": {
			CN: "容器服务ACK",
			EN: "ACK",
		},
		"data factory": {
			CN: "数据工厂",
			EN: "Data Factory",
		},
		"database": {
			CN: "数据库",
			EN: "Database",
		},
		"datahub": {
			CN: "数据总线 DataHub",
			EN: "DataHub",
		},
		"datav": {
			CN: "DataV数据可视化",
			EN: "DataV",
		},
		"dcdn": {
			CN: "全站加速",
			EN: "DCDN",
		},
		"des": {
			CN: "数据加密服务",
			EN: "DES",
		},
		"dide": {
			CN: "大数据开发治理平台 DataWorks",
			EN: "Big data development and governance platform DataWorks",
		},
		"directconnect": {
			CN: "专线接入",
			EN: "DirectConnect",
		},
		"disk": {
			CN: "块存储",
			EN: "Block storage",
		},
		"dms": {
			CN: "数据管理",
			EN: "Data management",
		},
		"dns": {
			CN: "域名解析服务",
			EN: "DNS",
		},
		"domain": {
			CN: "域名",
			EN: "Domain",
		},
		"dts": {
			CN: "数据传输",
			EN: "DTS",
		},
		"dysms": {
			CN: "短信服务",
			EN: "SMS service",
		},
		"eci": {
			CN: "弹性容器实例 ECI",
			EN: "ECI",
		},
		"ecs": {
			CN: "云服务器 ECS",
			EN: "ECS",
		},
		"edge computing machine": {
			CN: "边缘计算机器",
			EN: "Edge Computing Machine",
		},
		"eip": {
			CN: "Eip带宽",
			EN: "EIP",
		},
		"elasticsearch": {
			CN: "检索分析服务 Elasticsearch版",
			EN: "Search Analysis Service Elasticsearch Edition",
		},
		"emapreduce": {
			CN: "开源大数据平台 E-MapReduce",
			EN: "Open source big data platform E-MapReduce",
		},
		"english composition correction": {
			CN: "英文作文批改",
			EN: "English Composition Correction",
		},
		"expressconnect": {
			CN: "高速通道",
			EN: "High speed channel",
		},
		"face recognition": {
			CN: "人脸识别",
			EN: "Face Recognition",
		},
		"fc": {
			CN: "函数计算",
			EN: "Fc",
		},
		"file storage": {
			CN: "文件存储",
			EN: "File Storage",
		},
		"flowbag": {
			CN: "共享流量包",
			EN: "Shared traffic package",
		},
		"gaap": {
			CN: "全球应用加速",
			EN: "GAAP",
		},
		"gallery": {
			CN: "共享映像库",
			EN: "Gallery",
		},
		"gws": {
			CN: "无影云桌面",
			EN: "Shadowless Cloud Desktop",
		},
		"hbr": {
			CN: "混合云备份服务",
			EN: "Hybrid cloud backup service",
		},
		"hbrpost": {
			CN: "混合云备份",
			EN: "Hbrpost",
		},
		"hdm": {
			CN: "数据库自治服务",
			EN: "Database Autonomy Service",
		},
		"hitsdb": {
			CN: "时序数据库 InfluxDB® 版",
			EN: "Hitsdb",
		},
		"hologram": {
			CN: "实时数仓Hologres",
			EN: "Hologres",
		},
		"host": {
			CN: "专用宿主机",
			EN: "Host",
		},
		"idaas": {
			CN: "应用身份服务",
			EN: "Idaas",
		},
		"image": {
			CN: "镜像服务",
			EN: "Image",
		},
		"imm": {
			CN: "智能媒体管理",
			EN: "Intelligent Media Management",
		},
		"instant messaging": {
			CN: "即时通信",
			EN: "Instant Messaging",
		},
		"ipv6gateway": {
			CN: "IPv6 网关",
			EN: "IPv6 Gateway",
		},
		"kafka": {
			CN: "消息服务Kafka",
			EN: "Kafka",
		},
		"keymanager": {
			CN: "Key-Management-Service",
			EN: "KeyManager",
		},
		"kms": {
			CN: "密钥管理服务",
			EN: "Key management service",
		},
		"kvstore": {
			CN: "云数据库 Redis 版",
			EN: "ApsaraDB for Redis",
		},
		"lambda": {
			CN: "AWS-Lambda",
			EN: "Lambda",
		},
		"lb": {
			CN: "负载均衡",
			EN: "LB",
		},
		"live": {
			CN: "视频直播",
			EN: "Live video",
		},
		"live video broadcasting": {
			CN: "移动直播连麦",
			EN: "Live Video Broadcasting",
		},
		"log": {
			CN: "日志服务",
			EN: "Log",
		},
		"lvwang": {
			CN: "内容安全",
			EN: "Content security",
		},
		"machine learning": {
			CN: "机器学习",
			EN: "Machine Learning",
		},
		"mapreduce": {
			CN: "MapReduce",
			EN: "MapReduce",
		},
		"market": {
			CN: "云市场",
			EN: "Market",
		},
		"mem": {
			CN: "虚拟机内存",
			EN: "Vminstance Memory",
		},
		"mongodb": {
			CN: "云数据库MongoDB",
			EN: "MongoDB",
		},
		"monitor": {
			CN: "监控",
			EN: "Monitor",
		},
		"mqs": {
			CN: "消息队列服务",
			EN: "MQS",
		},
		"mse": {
			CN: "微服务引擎 MSE ",
			EN: "MSE ",
		},
		"msg": {
			CN: "消息通知服务",
			EN: "MSG",
		},
		"nas": {
			CN: "文件存储NAS",
			EN: "File Storage NAS",
		},
		"nat": {
			CN: "NAT网关",
			EN: "NAT",
		},
		"notification": {
			CN: "Notification",
			EN: "Notification",
		},
		"ntr": {
			CN: "转发路由器",
			EN: "NTR",
		},
		"ocr": {
			CN: "文字识别",
			EN: "OCR",
		},
		"odps": {
			CN: "云原生大数据计算服务 MaxCompute",
			EN: "MaxCompute",
		},
		"ons": {
			CN: "消息队列 RabbitMQ 版",
			EN: "Ons",
		},
		"oss": {
			CN: "对象存储",
			EN: "OSS",
		},
		"other": {
			CN: "其他服务",
			EN: "Other Services",
		},
		"polardb": {
			CN: "云原生关系型数据库 PolarDB",
			EN: "PolarDB",
		},
		"premiumsupport": {
			CN: "Premium-Support",
			EN: "PremiumSupport",
		},
		"prometheus": {
			CN: "Prometheus监控服务",
			EN: "Prometheus",
		},
		"pvtz": {
			CN: "内网DNS解析",
			EN: "DNS Private Zone",
		},
		"quickbi": {
			CN: "敏捷商业智能报表",
			EN: "Quickbi",
		},
		"rds": {
			CN: "关系型数据库",
			EN: "Relational Database",
		},
		"rds_disk": {
			CN: "RDS磁盘",
			EN: "RDS Disk",
		},
		"real time communication": {
			CN: "实时音视频",
			EN: "Real Time Communication",
		},
		"redshift": {
			CN: "Redshift",
			EN: "Redshift",
		},
		"reseller": {
			CN: "转销",
			EN: "Reseller",
		},
		"ri": {
			CN: "预留实例",
			EN: "RI",
		},
		"rounding": {
			CN: "计费精度差异",
			EN: "Rounding",
		},
		"saf": {
			CN: "风险识别",
			EN: "Risk identification",
		},
		"sc": {
			CN: "flink全托管",
			EN: "SC",
		},
		"scalinggroup": {
			CN: "弹性伸缩组",
			EN: "ScalingGroup",
		},
		"security center": {
			CN: "安全中心",
			EN: "Security Center",
		},
		"server": {
			CN: "虚拟机",
			EN: "Server",
		},
		"serversecurity": {
			CN: "主机安全",
			EN: "ServerSecurity",
		},
		"servicestage": {
			CN: "应用管理与运维平台",
			EN: "ServiceStage",
		},
		"sfs": {
			CN: "弹性文件服务",
			EN: "SFS",
		},
		"slb": {
			CN: "负载均衡",
			EN: "Load balancing",
		},
		"sls": {
			CN: "日志服务",
			EN: "Sls",
		},
		"smart oral evaluation": {
			CN: "口语评测",
			EN: "Smart Oral Evaluation",
		},
		"smartag": {
			CN: "智能接入网关APP",
			EN: "Smartag",
		},
		"sms": {
			CN: "短信",
			EN: "SMS",
		},
		"snapshot": {
			CN: "快照",
			EN: "Snapshot",
		},
		"sourcerepo": {
			CN: "SourceRepo",
			EN: "SourceRepo",
		},
		"speech recognition": {
			CN: "语音识别",
			EN: "Speech Recognition",
		},
		"ssl certificate": {
			CN: "SSL证书",
			EN: "SSL Certificate",
		},
		"storage": {
			CN: "存储",
			EN: "Storage",
		},
		"storage account": {
			CN: "azure对象存储",
			EN: "Storage Account",
		},
		"support": {
			CN: "Support",
			EN: "Support",
		},
		"t-sec": {
			CN: "T-Sec",
			EN: "T-Sec",
		},
		"t-sec anti ddos": {
			CN: "T-Sec DDoS",
			EN: "T-Sec Anti DDoS",
		},
		"t-sec cwp": {
			CN: "T-Sec主机安全",
			EN: "T-Sec CWP",
		},
		"tax": {
			CN: "税金",
			EN: "Tax",
		},
		"translate": {
			CN: "翻译服务",
			EN: "Translate",
		},
		"tsdb": {
			CN: "时序数据库",
			EN: "TSDB",
		},
		"vault": {
			CN: "保管库",
			EN: "Vault",
		},
		"video on demand": {
			CN: "点播",
			EN: "Video On Demand",
		},
		"vod": {
			CN: "视频点播",
			EN: "Video on demand",
		},
		"voice message": {
			CN: "语音消息",
			EN: "Voice Message",
		},
		"vpc": {
			CN: "VPC",
			EN: "VPC",
		},
		"vpn": {
			CN: "VPN网关",
			EN: "VPN",
		},
		"vpn gateway": {
			CN: "VPN网关",
			EN: "VPN Gateway",
		},
		"waf": {
			CN: "Web应用防火墙",
			EN: "Web application firewall",
		},
		"workflow": {
			CN: "工作流",
			EN: "Workflow",
		},
		"xtrace": {
			CN: "链路追踪",
			EN: "Link Tracking",
		},
	}
	for resource, translate := range resourceTranslate {
		ResourceTranslateI18nTable.Set(resource, i18n.NewTableEntry().CN(translate.CN).EN(translate.EN))
	}
}
