package sqlchemy

const (
	SQL_OP_AND      = "AND"
	SQL_OP_OR       = "OR"
	SQL_OP_NOT      = "NOT"
	SQL_OP_LIKE     = "LIKE"
	SQL_OP_IN       = "IN"
	SQL_OP_EQUAL    = "="
	SQL_OP_LT       = "<"
	SQL_OP_LE       = "<="
	SQL_OP_GT       = ">"
	SQL_OP_GE       = ">="
	SQL_OP_BETWEEN  = "BETWEEN"
	SQL_OP_NOTEQUAL = "<>"
)

const (
	TAG_IGNORE           = "ignore"
	TAG_NAME             = "name"
	TAG_WIDTH            = "width"
	TAG_TEXT_LENGTH      = "length"
	TAG_CHARSET          = "charset"
	TAG_PRECISION        = "precision"
	TAG_DEFAULT          = "default"
	TAG_UNIQUE           = "unique"
	TAG_INDEX            = "index"
	TAG_PRIMARY          = "primary"
	TAG_NULLABLE         = "nullable"
	TAG_AUTOINCREMENT    = "auto_increment"
	TAG_AUTOVERSION      = "auto_version"
	TAG_UPDATE_TIMESTAMP = "updated_at"
	TAG_CREATE_TIMESTAMP = "created_at"
	TAG_KEY_INDEX        = "key_index"
)
