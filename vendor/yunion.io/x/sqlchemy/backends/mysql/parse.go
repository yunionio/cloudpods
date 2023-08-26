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

package mysql

import (
	"regexp"

	"yunion.io/x/sqlchemy"
)

const (
	indexPattern      = `(?P<unique>UNIQUE\s+)?KEY ` + "`" + `(?P<name>\w+)` + "`" + ` \((?P<cols>` + "`" + `\w+` + "`" + `(\(\d+\))?(,\s*` + "`" + `\w+` + "`" + `(\(\d+\))?)*)\)`
	constraintPattern = `CONSTRAINT ` + "`" + `(?P<name>\w+)` + "`" + ` FOREIGN KEY \((?P<cols>` + "`" + `\w+` + "`" + `(,\s*` + "`" + `\w+` + "`" + `)*)\) REFERENCES ` + "`" + `(?P<table>\w+)` + "`" + ` \((?P<fcols>` + "`" + `\w+` + "`" + `(,\s*` + "`" + `\w+` + "`" + `)*)\)`
)

var (
	indexRegexp      = regexp.MustCompile(indexPattern)
	constraintRegexp = regexp.MustCompile(constraintPattern)
)

func fetchColumns(match string) []string {
	return sqlchemy.FetchColumns(match)
}

func parseConstraints(defStr string) []sqlchemy.STableConstraint {
	matches := constraintRegexp.FindAllStringSubmatch(defStr, -1)
	tcs := make([]sqlchemy.STableConstraint, len(matches))
	for i := range matches {
		tcs[i] = sqlchemy.NewTableConstraint(
			matches[i][1],
			fetchColumns(matches[i][2]),
			matches[i][4],
			fetchColumns(matches[i][5]),
		)
	}
	return tcs
}

func parseIndexes(ts sqlchemy.ITableSpec, defStr string) []sqlchemy.STableIndex {
	matches := indexRegexp.FindAllStringSubmatch(defStr, -1)
	tcs := make([]sqlchemy.STableIndex, len(matches))
	for i := range matches {
		tcs[i] = sqlchemy.NewTableIndex(
			ts,
			matches[i][2],
			fetchColumns(matches[i][3]),
			len(matches[i][1]) > 0,
		)
	}
	return tcs
}
