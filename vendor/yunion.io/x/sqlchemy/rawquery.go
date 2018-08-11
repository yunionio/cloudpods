package sqlchemy

type SRawQueryField struct {
	name string
}

func (rqf *SRawQueryField) Expression() string {
	return rqf.name
}

func (rqf *SRawQueryField) Name() string {
	return rqf.name
}

func (rqf *SRawQueryField) Reference() string {
	return rqf.name
}

func (rqf *SRawQueryField) Label(label string) IQueryField {
	return rqf
}

func NewRawQuery(sqlStr string, fields ...string) *SQuery {
	qfs := make([]IQueryField, len(fields))
	for i, f := range fields {
		rqf := SRawQueryField{name: f}
		qfs[i] = &rqf
	}
	q := SQuery{rawSql: sqlStr, fields: qfs}
	return &q
}
