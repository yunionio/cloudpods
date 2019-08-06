package compute

const (
	ELASTIC_CACHE_STATUS_RUNNING               = "running"               //（正常）
	ELASTIC_CACHE_STATUS_DEPLOYING             = "deploying"             //（创建中）
	ELASTIC_CACHE_STATUS_CHANGING              = "changing"              //（修改中）
	ELASTIC_CACHE_STATUS_INACTIVE              = "inactive"              //（被禁用）
	ELASTIC_CACHE_STATUS_FLUSHING              = "flushing"              //（清除中）
	ELASTIC_CACHE_STATUS_RELEASED              = "released"              //（已释放）
	ELASTIC_CACHE_STATUS_TRANSFORMING          = "transforming"          //（转换中）
	ELASTIC_CACHE_STATUS_UNAVAILABLE           = "unavailable"           //（服务停止）
	ELASTIC_CACHE_STATUS_ERROR                 = "error"                 //（创建失败）
	ELASTIC_CACHE_STATUS_MIGRATING             = "migrating"             //（迁移中）
	ELASTIC_CACHE_STATUS_BACKUPRECOVERING      = "backuprecovering"      //（备份恢复中）
	ELASTIC_CACHE_STATUS_MINORVERSIONUPGRADING = "minorversionupgrading" //（小版本升级中）
	ELASTIC_CACHE_STATUS_NETWORKMODIFYING      = "networkmodifying"      //（网络变更中）
	ELASTIC_CACHE_STATUS_SSLMODIFYING          = "sslmodifying"          //（SSL变更中）
	ELASTIC_CACHE_STATUS_MAJORVERSIONUPGRADING = "majorversionupgrading" //（大版本升级中，可正常访问）
)

const (
	ELASTIC_CACHE_ARCH_TYPE_STAND_ALONE  = "standalone"   //
	ELASTIC_CACHE_ARCH_TYPE_MASTER_SLAVE = "master_slave" //
	ELASTIC_CACHE_ARCH_TYPE_CLUSTER      = "cluster"      // 集群
	ELASTIC_CACHE_ARCH_TYPE_PROXY        = "proxy"        // 代理集群
)
