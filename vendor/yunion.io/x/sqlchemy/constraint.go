package sqlchemy

import (
	"regexp"
	"strings"

)

type STableConstraint struct {
	name string
	columns []string
	foreignTable string
	foreignKeys []string
}

const (
	indexPattern = `(?P<unique>UNIQUE\s+)?KEY ` + "`" + `(?P<name>\w+)` + "`" + ` \((?P<cols>` + "`" + `\w+` + "`" + `(\(\d+\))?(,\s*` + "`" + `\w+` + "`" + `(\(\d+\))?)*)\)`
	constraintPattern = `CONSTRAINT ` + "`" + `(?P<name>\w+)` + "`" + ` FOREIGN KEY \((?P<cols>` + "`" + `\w+` + "`" + `(,\s*` + "`" + `\w+` + "`" + `)*)\) REFERENCES ` + "`" + `(?P<table>\w+)` + "`" + ` \((?P<fcols>` + "`" + `\w+` + "`" + `(,\s*` + "`" + `\w+` + "`" + `)*)\)`
)

var (
	indexRegexp = regexp.MustCompile(indexPattern)
	constraintRegexp = regexp.MustCompile(constraintPattern)
)

func fetchColumns(match string) []string {
	ret := make([]string, 0)
	if len(match) > 0 {
		for _, part := range strings.Split(match, ",") {
			if part[len(part)-1] == ')' {
				part = part[:strings.LastIndexByte(part, '(')]
			}
			part = strings.Trim(part, "`")
			if len(part) > 0 {
				ret = append(ret, part)
			}
		}
	}
	// log.Debugf("%s", ret)
	return ret
}

func parseConstraints(defStr string) []STableConstraint {
	matches := constraintRegexp.FindAllStringSubmatch(defStr, -1)
	tcs := make([]STableConstraint, len(matches))
	for i := range matches {
		tcs[i] = STableConstraint{
			name: matches[i][1],
			foreignTable: matches[i][4],
			columns: fetchColumns(matches[i][2]),
			foreignKeys: fetchColumns(matches[i][5]),
		}
	}
	return tcs
}

func parseIndexes(defStr string) []STableIndex {
	matches := indexRegexp.FindAllStringSubmatch(defStr, -1)
	tcs := make([]STableIndex, len(matches))
	for i := range matches {
		tcs[i] = STableIndex{
			name: matches[i][2],
			isUnique: len(matches[i][1]) > 0,
			columns: fetchColumns(matches[i][3]),
		}
	}
	return tcs
}