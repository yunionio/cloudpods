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

package clickhouse

const (
	// TAG_PARTITION defines expression of PARTITION BY
	TAG_PARTITION = "clickhouse_partition_by"

	// TAG_ORDER defines fields of ORDER BY
	TAG_ORDER = "clickhouse_order_by"

	// TAG_TTL defines table TTL
	TAG_TTL = "clickhouse_ttl"

	EXTRA_OPTION_ENGINE_KEY             = "clickhouse_engine"
	EXTRA_OPTION_ENGINE_VALUE_MERGETRUE = "MergeTree"
	EXTRA_OPTION_ENGINE_VALUE_MYSQL     = "MySQL"

	// 'host:port', 'database', 'table', 'user', 'password'
	EXTRA_OPTION_CLICKHOUSE_MYSQL_HOSTPORT_KEY = "clickhouse_mysql_hostport"
	EXTRA_OPTION_CLICKHOUSE_MYSQL_DATABASE_KEY = "clickhouse_mysql_database"
	EXTRA_OPTION_CLICKHOUSE_MYSQL_TABLE_KEY    = "clickhouse_mysql_table"
	EXTRA_OPTION_CLICKHOUSE_MYSQL_USERNAME_KEY = "clickhouse_mysql_username"
	EXTRA_OPTION_CLICKHOUSE_MYSQL_PASSWORD_KEY = "clickhouse_mysql_password"
)
