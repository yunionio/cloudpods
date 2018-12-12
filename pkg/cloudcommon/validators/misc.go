package validators

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"
)

type ModelFilterOptions struct {
	Key          string
	ModelKeyword string
	ProjectId    string
}

func ApplyModelFilters(q *sqlchemy.SQuery, data *jsonutils.JSONDict, opts []*ModelFilterOptions) (*sqlchemy.SQuery, error) {
	var err error
	for _, opt := range opts {
		v := NewModelIdOrNameValidator(
			opt.Key,
			opt.ModelKeyword,
			opt.ProjectId,
		)
		v.Optional(true)
		q, err = v.QueryFilter(q, data)
		if err != nil {
			return nil, err
		}
	}
	return q, nil
}
