package types

type IRow interface {
	OvsdbTableName() string
	OvsdbIsRoot() bool
	OvsdbUuid() string

	SetColumn(name string, val interface{}) error

	OvsdbCmdArgs() []string

	HasExternalIds() bool
	SetExternalId(k, v string)
	GetExternalId(k string) (string, bool)
	RemoveExternalId(k string) (string, bool)
}

type ITable interface {
	OvsdbTableName() string
	OvsdbIsRoot() bool

	Rows() []IRow
	NewRow() IRow
	AppendRow(IRow)
}
