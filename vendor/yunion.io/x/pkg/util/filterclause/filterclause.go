package filterclause

import (
	"fmt"
	"regexp"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

type SFilterClause struct {
	field    string
	funcName string
	params   []string
}

// "guestnetworks.guest_id(id).ip_addr.equals(10.168.222.232)"
type SJointFilterClause struct {
	SFilterClause
	JointModel string
	RelatedKey string
	OriginKey  string
}

func (jfc *SJointFilterClause) GetJointFilter(q *sqlchemy.SQuery) sqlchemy.ICondition {
	return jfc.QueryCondition(q)
}

func (jfc *SJointFilterClause) GetJointModelName() string {
	return jfc.JointModel[:len(jfc.JointModel)-1]
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
	case "ge":
		return sqlchemy.GE(field, fc.params[0])
	case "gt":
		return sqlchemy.GT(field, fc.params[0])
	case "le":
		return sqlchemy.LE(field, fc.params[0])
	case "lt":
		return sqlchemy.LT(field, fc.params[0])
	case "like":
		return sqlchemy.Like(field, fc.params[0])
	case "contains":
		return sqlchemy.Contains(field, fc.params[0])
	case "startswith":
		return sqlchemy.Startswith(field, fc.params[0])
	case "endswith":
		return sqlchemy.Endswith(field, fc.params[0])
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
	case "isnullorempty":
		return sqlchemy.IsNullOrEmpty(field)
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
	filterClausePattern      *regexp.Regexp
	jointFilterClausePattern *regexp.Regexp
)

func init() {
	filterClausePattern = regexp.MustCompile(`^(\w+)\.(\w+)\((.*)\)`)
	jointFilterClausePattern = regexp.MustCompile(`^(\w+)\.(\w+)\((\w+)\).(\w+)\.(\w+)\((.*)\)`)
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

func ParseJointFilterClause(jointFilter string) *SJointFilterClause {
	matches := jointFilterClausePattern.FindStringSubmatch(jointFilter)
	if matches == nil {
		return nil
	}
	params := utils.FindWords([]byte(matches[6]), 0)
	jfc := SJointFilterClause{
		SFilterClause: SFilterClause{
			field:    matches[4],
			funcName: matches[5],
			params:   params,
		},
		JointModel: matches[1],
		RelatedKey: matches[2],
		OriginKey:  matches[3],
	}
	return &jfc
}
