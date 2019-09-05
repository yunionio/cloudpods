package db

import (
	"yunion.io/x/sqlchemy"
)

type SPagingConfig struct {
	Order        sqlchemy.QueryOrderType
	MarkerField  string
	DefaultLimit int
}
