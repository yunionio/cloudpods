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

package compute

const (
	//实例状态
	DBINSTANCE_INIT                  = "init"              //初始化
	DBINSTANCE_DEPLOYING             = "deploying"         //部署中
	DBINSTANCE_RUNNING               = "running"           //运行中
	DBINSTANCE_REBOOTING             = "rebooting"         //重启中
	DBINSTANCE_MIGRATING             = "migrating"         //迁移中
	DBINSTANCE_BACKING_UP            = "backing_up"        //备份中
	DBINSTANCE_BACKING_UP_FAILED     = "backing_up_failed" //备份失败
	DBINSTANCE_RESTORING             = "restoring"         //备份恢复中
	DBINSTANCE_RESTORE_FAILED        = "restore_failed"
	DBINSTANCE_IMPORTING             = "importing"             //数据导入中
	DBINSTANCE_CLONING               = "cloning"               //克隆中
	DBINSTANCE_DELETING              = "deleting"              //删除中
	DBINSTANCE_DELETE_FAILED         = "delete_failed"         //删除失败
	DBINSTANCE_MAINTENANCE           = "maintenance"           //维护中
	DBINSTANCE_ISOLATING             = "isolating"             //隔离中
	DBINSTANCE_ISOLATE               = "isolate"               //已隔离
	DBINSTANCE_UPGRADING             = "upgrading"             //升级中
	DBINSTANCE_SET_AUTO_RENEW        = "set_auto_renew"        //设置自动续费中
	DBINSTANCE_SET_AUTO_RENEW_FAILED = "set_auto_renew_failed" //设置自动续费失败
	DBINSTANCE_UNKNOWN               = "unknown"
	DBINSTANCE_SYNC_SECGROUP_FAILED  = "sync_secgroup_failed" // 同步安全组失败

	DBINSTANCE_CHANGE_CONFIG        = "change_config"        //调整配置
	DBINSTANCE_CHANGE_CONFIG_FAILED = "change_config_failed" //调整配置失败

	DBINSTANCE_RENEWING     = "renewing"     //续费中
	DBINSTANCE_RENEW_FAILED = "renew_failed" //续费失败

	DBINSTANCE_SYNC_CONFIG = "sync_config" //同步配置

	DBINSTANCE_REBOOT_FAILED = "reboot_failed" //重启失败
	DBINSTANCE_CREATE_FAILED = "create_failed" //创建失败

	DBINSTANCE_FAILE = "failed" //操作失败

	DBINSTANCE_UPDATE_TAGS        = "update_tags"
	DBINSTANCE_UPDATE_TAGS_FAILED = "update_tags_fail"

	//备份状态
	DBINSTANCE_BACKUP_READY         = "ready"         //正常
	DBINSTANCE_BACKUP_CREATING      = "creating"      //创建中
	DBINSTANCE_BACKUP_CREATE_FAILED = "create_failed" //创建失败
	DBINSTANCE_BACKUP_DELETING      = "deleting"      //删除中
	DBINSTANCE_BACKUP_DELETE_FAILED = "delete_failed" //删除失败
	DBINSTANCE_BACKUP_FAILED        = "failed"        //异常
	DBINSTANCE_BACKUP_UNKNOWN       = "unknown"       //未知

	//备份模式
	BACKUP_MODE_AUTOMATED = "automated" //自动
	BACKUP_MODE_MANUAL    = "manual"    //手动

	//实例数据库状态
	DBINSTANCE_DATABASE_CREATING        = "creating"        //创建中
	DBINSTANCE_DATABASE_CREATE_FAILE    = "create_failed"   //创建失败
	DBINSTANCE_DATABASE_RUNNING         = "running"         //正常
	DBINSTANCE_DATABASE_GRANT_PRIVILEGE = "grant_privilege" //赋予权限中
	DBINSTANCE_DATABASE_DELETING        = "deleting"        //删除中
	DBINSTANCE_DATABASE_DELETE_FAILED   = "delete_failed"   //删除失败

	//实例用户状态
	DBINSTANCE_USER_UNAVAILABLE         = "unavailable"         //不可用
	DBINSTANCE_USER_AVAILABLE           = "available"           //正常
	DBINSTANCE_USER_CREATING            = "creating"            //创建中
	DBINSTANCE_USER_CREATE_FAILED       = "create_failed"       //创建失败
	DBINSTANCE_USER_DELETING            = "deleting"            //删除中
	DBINSTANCE_USER_DELETE_FAILED       = "delete_failed"       //删除失败
	DBINSTANCE_USER_RESET_PASSWD        = "reset_passwd"        //重置密码中
	DBINSTANCE_USER_GRANT_PRIVILEGE     = "grant_privilege"     //赋予权限中
	DBINSTANCE_USER_SET_PRIVILEGE       = "set_privilege"       //设置权限中
	DBINSTANCE_USER_REVOKE_PRIVILEGE    = "revoke_privilege"    //解除权限中
	DBINSTANCE_USER_RESET_PASSWD_FAILED = "reset_passwd_failed" //重置密码失败

	//数据库权限
	DATABASE_PRIVILEGE_RW     = "rw" //读写
	DATABASE_PRIVILEGE_R      = "r"  //只读
	DATABASE_PRIVILEGE_DDL    = "ddl"
	DATABASE_PRIVILEGE_DML    = "dml"
	DATABASE_PRIVILEGE_OWNER  = "owner"
	DATABASE_PRIVILEGE_CUSTOM = "custom" //自定义

	DBINSTANCE_TYPE_MYSQL      = "MySQL"
	DBINSTANCE_TYPE_SQLSERVER  = "SQLServer"
	DBINSTANCE_TYPE_POSTGRESQL = "PostgreSQL"
	DBINSTANCE_TYPE_MARIADB    = "MariaDB"
	DBINSTANCE_TYPE_ORACLE     = "Oracle"
	DBINSTANCE_TYPE_PPAS       = "PPAS"
	DBINSTANCE_TYPE_PERCONA    = "Percona"

	//阿里云实例类型
	ALIYUN_DBINSTANCE_CATEGORY_BASIC    = "basic"             //基础版
	ALIYUN_DBINSTANCE_CATEGORY_HA       = "high_availability" //高可用
	ALIYUN_DBINSTANCE_CATEGORY_ALWAYSON = "always_on"         //集群版
	ALIYUN_DBINSTANCE_CATEGORY_FINANCE  = "finance"           //金融版

	//腾讯云实例类型
	QCLOUD_DBINSTANCE_CATEGORY_BASIC   = "basic" //基础版
	QCLOUD_DBINSTANCE_CATEGORY_HA      = "ha"    //高可用
	QCLOUD_DBINSTANCE_CATEGORY_FINANCE = "fe"    //金融版
	QCLOUD_DBINSTANCE_CATEGORY_TDSQL   = "tdsql" //TDSQL

	//华为云实例类型
	HUAWEI_DBINSTANCE_CATEGORY_HA      = "ha"      //主备
	HUAWEI_DBINSTANCE_CATEGORY_SINGLE  = "single"  //单机
	HUAWEI_DBINSTANCE_CATEGORY_REPLICA = "replica" //只读

	//谷歌云实例类型
	GOOGLE_DBINSTANCE_CATEGORY_REGIONAL = "Regional" // 高可用性（区域级）
	GOOGLE_DBINSTANCE_CATEGORY_ZONAL    = "Zonal"    // 单个地区

	// Azure
	AZURE_DBINSTANCE_CATEGORY_BASIC = "basic"

	//阿里云存储类型
	ALIYUN_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD  = "local_ssd"  //本地盘SSD盘
	ALIYUN_DBINSTANCE_STORAGE_TYPE_CLOUD_ESSD = "cloud_essd" //ESSD云盘
	ALIYUN_DBINSTANCE_STORAGE_TYPE_CLOUD_SSD  = "cloud_ssd"  //SSD云盘

	//华为云存储类型
	HUAWEI_DBINSTANCE_STORAGE_TYPE_ULTRAHIGH    = "ULTRAHIGH" //超高IO云硬盘
	HUAWEI_DBINSTANCE_STORAGE_TYPE_ULTRAHIGHPRO = "ULTRAHIGHPRO"
	HUAWEI_DBINSTANCE_STORAGE_TYPE_COMMON       = "COMMON"
	HUAWEI_DBINSTANCE_STORAGE_TYPE_HIGH         = "HIGH"

	//腾讯云
	QCLOUD_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD = "local_ssd" //本地盘SSD盘
	QCLOUD_DBINSTANCE_STORAGE_TYPE_CLOUD_SSD = "cloud_ssd" //SSD云盘

)

var (
	ALIYUN_MYSQL_DENY_KEYWORKD []string = []string{
		"accessible", "account", "action", "actual", "add", "after", "against", "aggregate", "all", "algorithm", "alter", "always", "analyse", "analyze", "and", "any", "as", "asc", "ascii", "asensitive", "at", "auto_increment", "autoextend_size", "avg", "avg_row_length", "backup", "before", "begin", "between", "bigint", "binary", "binlog", "bit", "blob", "block", "bool", "boolean", "both", "btree", "by", "byte", "cache", "call", "cascade", "cascaded", "case", "catalog_name", "chain", "change", "changed", "channel", "char", "character", "charset", "check", "checksum", "cipher", "class_origin", "client", "close", "coalesce", "code", "collate", "collation", "column", "column_format", "column_name", "columns", "comment", "commit", "committed", "compact", "completion", "compression", "compressed", "encryption", "concurrent", "condition", "connection", "consistent", "constraint", "constraint_catalog", "constraint_name", "constraint_schema", "contains", "context", "continue", "convert", "cpu", "create", "cross", "cube", "current", "current_date", "current_time", "current_timestamp", "current_user", "cursor", "cursor_name", "data", "database", "databases", "datafile", "date", "datetime", "day", "day_hour", "day_microsecond", "day_minute", "day_second", "deallocate", "dec", "decimal", "declare", "default", "default_auth", "definer", "delayed", "delay_key_write", "desc", "describe", "des_key_file", "deterministic", "diagnostics", "directory", "disable", "discard", "disk", "distinct", "distinctrow", "div", "do", "double", "drop", "dual", "dumpfile", "duplicate", "dynamic", "each", "else", "elseif", "enable", "enclosed", "end", "ends", "engine", "engines", "enum", "error", "errors", "escape", "escaped", "event", "events", "every", "exchange", "execute", "exists", "exit", "expansion", "export", "expire", "explain", "extended", "extent_size", "false", "fast", "faults", "fetch", "fields", "file", "file_block_size", "filter", "first", "fixed", "float", "float4", "float8", "flush", "follows", "for", "force", "foreign", "format", "found", "from", "full", "fulltext", "function", "general", "group_replication", "geometry", "geometrycollection", "get_format", "get", "generated", "global", "grant", "grants", "group", "handler", "hash", "having", "help", "high_priority", "host", "hosts", "hour", "hour_microsecond", "hour_minute", "hour_second", "identified", "if", "ignore", "ignore_server_ids", "import", "in", "index", "indexes", "infile", "initial_size", "inner", "inout", "insensitive", "insert_method", "install", "instance", "int", "int1", "int2", "int3", "int4", "int8", "integer", "interval", "into", "io", "io_after_gtids", "io_before_gtids", "io_thread", "ipc", "is", "isolation", "issuer", "iterate", "invoker", "join", "json", "key", "keys", "key_block_size", "kill", "language", "last", "leading", "leave", "leaves", "left", "less", "level", "like", "limit", "linear", "lines", "linestring", "list", "load", "local", "localtime", "localtimestamp", "lock", "locks", "logfile", "logs", "long", "longblob", "longtext", "loop", "low_priority", "master", "master_auto_position", "master_bind", "master_connect_retry", "master_delay", "master_host", "master_log_file", "master_log_pos", "master_password", "master_port", "master_retry_count", "master_server_id", "master_ssl", "master_ssl_ca", "master_ssl_capath", "master_tls_version", "master_ssl_cert", "master_ssl_cipher", "master_ssl_crl", "master_ssl_crlpath", "master_ssl_key", "master_ssl_verify_server_cert", "master_user", "master_heartbeat_period", "match", "max_connections_per_hour", "max_queries_per_hour", "max_rows", "max_size", "max_updates_per_hour", "max_user_connections", "maxvalue", "medium", "mediumblob", "mediumint", "mediumtext", "memory", "merge", "message_text", "microsecond", "middleint", "migrate", "minute", "minute_microsecond", "minute_second", "min_rows", "mod", "mode", "modifies", "modify", "month", "multilinestring", "multipoint", "multipolygon", "mutex", "mysql_errno", "name", "names", "national", "natural", "ndb", "ndbcluster", "nchar", "never", "new", "next", "no", "no_wait", "nodegroup", "none", "not", "no_write_to_binlog", "null", "number", "numeric", "nvarchar", "offset", "on", "one", "only", "open", "optimize", "optimizer_costs", "options", "option", "optionally", "or", "order", "out", "outer", "outfile", "owner", "pack_keys", "parser", "page", "parse_gcol_expr", "partial", "partition", "partitioning", "partitions", "password", "phase", "plugin", "plugins", "plugin_dir", "point", "polygon", "port", "precedes", "precision", "prepare", "preserve", "prev", "primary", "privileges", "procedure", "process", "processlist", "profile", "profiles", "proxy", "purge", "quarter", "query", "quick", "range", "rds_charsetnum", "rds_connection_id", "rds_prepare_begin_id", "rds_current_connection", "rds_db", "rds_rw_mode", "rds_host_show", "rds_resetconnection", "read", "read_only", "read_write", "reads", "real", "rebuild", "recover", "redo_buffer_size", "redofile", "redundant", "references", "regexp", "relay", "relaylog", "relay_log_file", "relay_log_pos", "relay_thread", "release", "reload", "remove", "rename", "reorganize", "repair", "repeatable", "replication", "replicate_do_db", "replicate_ignore_db", "replicate_do_table", "replicate_ignore_table", "replicate_wild_do_table", "replicate_wild_ignore_table", "replicate_rewrite_db", "repeat", "require", "reset", "resignal", "restore", "restrict", "resume", "returned_sqlstate", "return", "returns", "reverse", "revoke", "right", "rlike", "rollback", "rollup", "routine", "rotate", "row", "row_count", "rows", "row_format", "rtree", "savepoint", "schedule", "schema", "schema_name", "schemas", "second", "second_microsecond", "security", "sensitive", "separator", "serial", "serializable", "session", "server", "set", "share", "show", "shutdown", "signal", "signed", "simple", "slave", "slow", "snapshot", "smallint", "socket", "some", "soname", "sounds", "source", "spatial", "specific", "sql", "sqlexception", "sqlstate", "sqlwarning", "sql_after_gtids", "sql_after_mts_gaps", "sql_before_gtids", "sql_big_result", "sql_buffer_result", "sql_cache", "sql_calc_found_rows", "sql_filters", "sql_no_cache", "sql_small_result", "sql_thread", "sql_tsi_second", "sql_tsi_minute", "sql_tsi_hour", "sql_tsi_day", "sql_tsi_week", "sql_tsi_month", "sql_tsi_quarter", "sql_tsi_year", "ssl", "stacked", "start", "starting", "starts", "stats_auto_recalc", "stats_persistent", "stats_sample_pages", "status", "stop", "storage", "stored", "straight_join", "string", "subclass_origin", "subject", "subpartition", "subpartitions", "super", "suspend", "swaps", "switches", "table", "table_name", "tables", "tablespace", "table_checksum", "temporary", "temptable", "terminated", "text", "than", "then", "time", "timestamp", "timestampadd", "timestampdiff", "tinyblob", "tinyint", "tinytext", "to", "trailing", "transaction", "trigger", "triggers", "true", "truncate", "type", "types", "uncommitted", "undefined", "undo_buffer_size", "undofile", "undo", "unicode", "union", "unique", "unknown", "unlock", "uninstall", "unsigned", "until", "upgrade", "usage", "use", "user", "user_resources", "use_frm", "using", "utc_date", "utc_time", "utc_timestamp", "validation", "value", "values", "varbinary", "varchar", "varcharacter", "variables", "varying", "wait", "warnings", "week", "weight_string", "when", "where", "while", "view", "virtual", "with", "without", "work", "wrapper", "write", "x509", "xor", "xa", "xid", "xml", "year", "year_month", "zerofill", "lag", "rds_audit", "rds_inner_backup", "rds_change_user", "rds_user", "rds_ip", "rds_add_proxy_protocol_networks", "rds_show_proxy_protocol_ips", "sync", "delete", "insert", "replace", "select", "update", "adddate", "bit_and", "bit_or", "bit_xor", "cast", "count", "curdate", "curtime", "date_add", "date_sub", "extract", "group_concat", "json_objectagg", "json_arrayagg", "max", "mid", "min", "now", "position", "session_user", "std", "stddev", "stddev_pop", "stddev_samp", "subdate", "substr", "substring", "sum", "sysdate", "system_user", "trim", "variance", "var_pop", "var_samp", "bka", "bnl", "dupsweedout", "firstmatch", "intoexists", "loosescan", "materialization", "max_execution_time", "no_bka", "no_bnl", "no_icp", "no_mrr", "no_range_optimization", "no_semijoin", "mrr", "qb_name", "semijoin", "subquery",
	}

	ALIYUN_SQL_SERVER_DENY_KEYWORD []string = []string{
		"root", " admin", " eagleye", " master", " aurora", " sa", " sysadmin", " administrator", " mssqld", " public", " securityadmin", " serveradmin", " setupadmin", " processadmin", " diskadmin", " dbcreator", " bulkadmin", " tempdb", " msdb", " model", " distribution", " mssqlsystemresource", " guest", " add", " except", " percent", " all", " exec", " plan", " alter", " execute", " precision", " and", " exists", " primary", " any", " exit", " print", " as", " fetch", " proc", " asc", " file", " procedure", " authorization", " fillfactor", " public", " backup", " for", " raiserror", " begin", " foreign", " read", " between", " freetext", " readtext", " break", " freetexttable", " reconfigure", " browse", " from", " references", " bulk", " full", " replication", " by", " function", " restore", " cascade", " goto", " restrict", " case", " grant", " return", " check", " group", " revoke", " checkpoint", " having", " right", " close", " holdlock", " rollback", " clustered", " identity", " rowcount", " coalesce", " identity_insert", " rowguidcol", " collate", " identitycol", " rule", " column", " if", " save", " commit", " in", " schema", " compute", " index", " select", " constraint", " inner", " session_user", " contains", " insert", " set", " containstable", " intersect", " setuser", " continue", " into", " shutdown", " convert", " is", " some", " create", " join", " statistics", " cross", " key", " system_user", " current", " kill", " table", " current_date", " left", " textsize", " current_time", " like", " then", " current_timestamp", " lineno", " to", " current_user", " load", " top", " cursor", " national", " tran", " database", " nocheck", " transaction", " dbcc", " nonclustered", " trigger", " deallocate", " not", " truncate", " declare", " null", " tsequal", " default", " nullif", " union", " delete", " of", " unique", " deny", " off", " update", " desc", " offsets", " updatetext", " disk", " on", " use", " distinct", " open", " user", " distributed", " opendatasource", " values", " double", " openquery", " varying", " drop", " openrowset", " view", " dummy", " openxml", " waitfor", " dump", " option", " when", " else", " or", " where", " end", " order", " while", " errlvl", " outer", " with", " escape", " over", " writetext", " galaxy",
	}

	RW_PRIVILEGE_SET = []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE",
		"DROP", "REFERENCES", "INDEX", "ALTER", "CREATE TEMPORARY TABLES",
		"LOCK TABLES", "EXECUTE", "CREATE VIEW", "SHOW VIEW", "CREATE ROUTINE",
		"ALTER ROUTINE", "EVENT", "TRIGGER", "PROCESS", "REPLICATION SLAVE",
		"REPLICATION CLIENT",
	}
	R_PRIVILEGE_SET         = []string{"SELECT", "LOCK TABLES", "SHOW VIEW", "PROCESS", "REPLICATION SLAVE", "REPLICATION CLIENT"}
	QCLOUD_RW_PRIVILEGE_SET = []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE",
		"DROP", "REFERENCES", "INDEX", "ALTER", "CREATE TEMPORARY TABLES",
		"LOCK TABLES", "EXECUTE", "CREATE VIEW", "SHOW VIEW", "CREATE ROUTINE",
		"ALTER ROUTINE", "EVENT", "TRIGGER",
	}
	QCLOUD_R_PRIVILEGE_SET = []string{"SELECT", "LOCK TABLES", "SHOW VIEW"}
)
