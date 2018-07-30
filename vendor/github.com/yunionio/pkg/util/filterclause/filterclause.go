package filterclause

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yunionio/log"
	"github.com/yunionio/pkg/utils"
	"github.com/yunionio/sqlchemy"
)

type SFilterClause struct {
	field    string
	funcName string
	params   []string
}

func (fc *SFilterClause) QueryCondition(q *sqlchemy.SQuery) sqlchemy.ICondition {
	field := q.Field(fc.field)
	if field == nil {
		log.Errorf("Cannot find filed %s", fc.field)
		return nil
	}
	switch fc.funcName {
	case "in":
		return sqlchemy.In(field, fc.params)
	case "notin":
		return sqlchemy.NotIn(field, fc.params)
	case "between":
		return sqlchemy.Between(field, fc.params[0], fc.params[1])
	case "like":
		return sqlchemy.Like(field, fc.params[0])
	case "contains":
		return sqlchemy.Like(field, fmt.Sprintf("%%%s%%", fc.params[0]))
	case "startswith":
		return sqlchemy.Like(field, fmt.Sprint("%s%%", fc.params[0]))
	case "endswith":
		return sqlchemy.Like(field, fmt.Sprintf("%%%s", fc.params[0]))
	case "equals":
		return sqlchemy.Equals(field, fc.params[0])
	case "notequals":
		return sqlchemy.NOT(sqlchemy.Equals(field, fc.params[0]))
	case "isnull":
		return sqlchemy.IsNull(field)
	case "isnotnull":
		return sqlchemy.IsNotNull(field)
	case "isempty":
		return sqlchemy.IsEmpty(field)
	case "isnotempty":
		return sqlchemy.IsNotEmpty(field)
	default:
		return nil
	}
}

func (fc *SFilterClause) GetField() string {
	return fc.field
}

func (fc *SFilterClause) String() string {
	return fmt.Sprintf("%s.%s(%s)", fc.field, fc.funcName, strings.Join(fc.params, ","))
}

var (
	filterClausePattern *regexp.Regexp
)

func init() {
	filterClausePattern = regexp.MustCompile(`^(\w+)\.(\w+)\((.*)\)`)
}

func ParseFilterClause(filter string) *SFilterClause {
	matches := filterClausePattern.FindStringSubmatch(filter)
	if matches == nil {
		return nil
	}
	params := utils.FindWords([]byte(matches[3]), 0)
	fc := SFilterClause{field: matches[1], funcName: matches[2], params: params}
	return &fc
}
