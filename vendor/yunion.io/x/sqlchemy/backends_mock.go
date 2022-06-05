package sqlchemy

import (
	"database/sql"
	"fmt"
	"reflect"

	"yunion.io/x/pkg/utils"
)

const (
	mockBackendName = DBBackendName("mock")
)

type sMockColumn struct {
	SBaseColumn
	isCreatedAt   bool
	isUpdatedAt   bool
	isAutoVersion bool
}

func (c *sMockColumn) DefinitionString() string {
	return fmt.Sprintf("%s %s", c.name, c.sqlType)
}

func (c *sMockColumn) ConvertFromString(str string) interface{} {
	return str
}

func (c *sMockColumn) ConvertFromValue(v interface{}) interface{} {
	return v
}

func (c *sMockColumn) IsAutoVersion() bool {
	return c.isAutoVersion
}

func (c *sMockColumn) IsUpdatedAt() bool {
	return c.isUpdatedAt
}

func (c *sMockColumn) IsCreatedAt() bool {
	return c.isCreatedAt
}

func (c *sMockColumn) IsNumeric() bool {
	switch c.sqlType {
	case "int", "uint", "int8", "int16", "int32", "int64", "uin8", "uint16", "uint32", "uint64", "float32", "float64":
		return true
	}
	return false
}

func (c *sMockColumn) IsText() bool {
	return c.sqlType == "string"
}

func (c *sMockColumn) IsZero(val interface{}) bool {
	return reflect.ValueOf(val).IsZero()
}

func newMockColumn(name string, sqlType string, tagMap map[string]string, isPointer bool) sMockColumn {
	var (
		isCreatedAt   = false
		isUpdatedAt   = false
		isAutoVersion = false
		val           string
		ok            bool
	)
	tagMap, val, ok = utils.TagPop(tagMap, TAG_CREATE_TIMESTAMP)
	if ok {
		isCreatedAt = utils.ToBool(val)
	}
	tagMap, val, ok = utils.TagPop(tagMap, TAG_UPDATE_TIMESTAMP)
	if ok {
		isUpdatedAt = utils.ToBool(val)
	}
	tagMap, val, ok = utils.TagPop(tagMap, TAG_AUTOVERSION)
	if ok {
		isAutoVersion = utils.ToBool(val)
	}
	return sMockColumn{
		SBaseColumn:   NewBaseColumn(name, sqlType, tagMap, isPointer),
		isCreatedAt:   isCreatedAt,
		isUpdatedAt:   isUpdatedAt,
		isAutoVersion: isAutoVersion,
	}
}

type sMockBackend struct {
	SBaseBackend
}

func SetupMockDatabaseBackend() {
	RegisterBackend(&sMockBackend{})

	SetDBWithNameBackend(&sql.DB{}, DefaultDB, mockBackendName)
}

func (mock *sMockBackend) Name() DBBackendName {
	return mockBackendName
}

// CanUpdate returns wether the backend supports update
func (mock *sMockBackend) CanUpdate() bool {
	return true
}

// CanInsert returns wether the backend supports Insert
func (mock *sMockBackend) CanInsert() bool {
	return true
}

// CanInsertOrUpdate returns weather the backend supports InsertOrUpdate
func (mock *sMockBackend) CanInsertOrUpdate() bool {
	return true
}

func (mock *sMockBackend) DropIndexSQLTemplate() string {
	return ""
}

func (mock *sMockBackend) InsertOrUpdateSQLTemplate() string {
	return "INSERT INTO `{{ .Table }}` ({{ .Columns }}) VALUES ({{ .Values }}) ON DUPLICATE KEY UPDATE {{ .SetValues }}"
}

func (mock *sMockBackend) GetTableSQL() string {
	return ""
}

func (mock *sMockBackend) IsSupportIndexAndContraints() bool {
	return true
}

func (mock *sMockBackend) GetCreateSQLs(ts ITableSpec) []string {
	return nil
}

func (mock *sMockBackend) FetchIndexesAndConstraints(ts ITableSpec) ([]STableIndex, []STableConstraint, error) {
	return nil, nil, nil
}

func (mock *sMockBackend) FetchTableColumnSpecs(ts ITableSpec) ([]IColumnSpec, error) {
	return nil, nil
}

func (mock *sMockBackend) GetColumnSpecByFieldType(table *STableSpec, fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) IColumnSpec {
	col := newMockColumn(fieldname, fieldType.String(), tagmap, isPointer)
	return &col
}

func (mock *sMockBackend) CurrentUTCTimeStampString() string {
	return "UTC_NOW()"
}

func (mock *sMockBackend) CurrentTimeStampString() string {
	return "NOW()"
}

func (mock *sMockBackend) CommitTableChangeSQL(ts ITableSpec, changes STableChanges) []string {
	return []string{}
}
