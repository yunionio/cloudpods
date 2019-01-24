package huawei

// 华为云返回的时间格式
const DATETIME_FORMAT = "2006-01-02T15:04:05.999999999"

// Task status
const TASK_SUCCESS = "SUCCESS"

// Charging Type
const (
	POST_PAID = "postPaid" // 按需付费
	PRE_PAID  = "prePaid"  // 包年包月
)

// 资源类型 https://support.huaweicloud.com/api-oce/zh-cn_topic_0079291752.html
const (
	RESOURCE_TYPE_VM        = "hws.resource.type.vm"          // ECS虚拟机
	RESOURCE_TYPE_VOLUME    = "hws.resource.type.volume"      // EVS卷
	RESOURCE_TYPE_BANDWIDTH = "hws.resource.type.bandwidth"   // VPC带宽
	RESOURCE_TYPE_IP        = "hws.resource.type.ip"          // VPC公网IP
	RESOURCE_TYPE_IMAGE     = "hws.resource.type.marketplace" // 市场镜像
)
