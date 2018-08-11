package sqlchemy

import (
	"yunion.io/x/log"
)

var (
	DEBUG_SQLCHEMY = false
)

func (tq *SQuery) DebugQuery() {
	sqlstr := tq.String()
	vars := tq.Variables()
	log.Debugf("SQuery %s with vars: %s", sqlstr, vars)
}

func (sqf *SSubQuery) DebugQuery() {
	sqlstr := sqf.query.String()
	vars := sqf.query.Variables()
	log.Debugf("SQuery %s with vars: %s", sqlstr, vars)
}

func (t *STableSpec) DebugInsert(dt interface{}) error {
	return t.insert(dt, true)
}

func (ts *STableSpec) DebugUpdateFields(dt interface{}, fields map[string]interface{}) error {
	return ts.updateFields(dt, fields, true)
}
