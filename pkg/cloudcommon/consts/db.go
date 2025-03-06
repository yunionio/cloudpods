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

package consts

import "yunion.io/x/log"

var (
	QueryOffsetOptimization = false

	OpsLogWithClickhouse = false

	defaultDBDialect string

	defaultDBConnectionString string

	defaultDBChecksumHashAlgorithm string

	taskWorkerCount      int
	localTaskWorkerCount int

	taskArchiveThresholdHours int

	taskArchiveBatchLimit int

	enableChangeOwnerAutoRename = false

	enableDefaultPolicy = true
)

func SetDefaultPolicy(enable bool) {
	enableDefaultPolicy = enable
}

func IsEnableDefaultPolicy() bool {
	return enableDefaultPolicy == true
}

func SetDefaultDB(dialect, connStr string) {
	defaultDBDialect = dialect
	defaultDBConnectionString = connStr
}

func DefaultDBDialect() string {
	return defaultDBDialect
}

func DefaultDBConnStr() string {
	return defaultDBConnectionString
}

func SetDefaultDBChecksumHashAlgorithm(alg string) {
	log.Infof("Set default DB checksum hash algorithm: %s", alg)
	defaultDBChecksumHashAlgorithm = alg
}

func DefaultDBChecksumHashAlgorithm() string {
	if len(defaultDBChecksumHashAlgorithm) > 0 {
		return defaultDBChecksumHashAlgorithm
	}
	return "sha256"
}

func SetTaskWorkerCount(cnt int) {
	taskWorkerCount = cnt
}

func SetLocalTaskWorkerCount(cnt int) {
	localTaskWorkerCount = cnt
}

func SetChangeOwnerAutoRename(enable bool) {
	enableChangeOwnerAutoRename = enable
}

func GetChangeOwnerAutoRename() bool {
	return enableChangeOwnerAutoRename
}

func SetTaskArchiveThresholdHours(hours int) {
	taskArchiveThresholdHours = hours
}

func SetTaskArchiveBatchLimit(limit int) {
	taskArchiveBatchLimit = limit
}

func TaskWorkerCount() int {
	return taskWorkerCount
}

func LocalTaskWorkerCount() int {
	return localTaskWorkerCount
}

func TaskArchiveThresholdHours() int {
	return taskArchiveThresholdHours
}

func TaskArchiveBatchLimit() int {
	return taskArchiveBatchLimit
}
