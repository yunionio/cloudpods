package mysql

import (
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/sqlchemy"
)

func init() {
	sqlchemy.RegisterBackend(&SMySQLBackend{})
}

type SMySQLBackend struct {
	sqlchemy.SBaseBackend
}

func (mysql *SMySQLBackend) Name() sqlchemy.DBBackendName {
	return sqlchemy.MySQLBackend
}

func (mysql *SMySQLBackend) GetCreateSQL(ts sqlchemy.ITableSpec) string {
	cols := make([]string, 0)
	primaries := make([]string, 0)
	indexes := make([]string, 0)
	autoInc := ""
	for _, c := range ts.Columns() {
		cols = append(cols, c.DefinitionString())
		if c.IsPrimary() {
			primaries = append(primaries, fmt.Sprintf("`%s`", c.Name()))
			if intC, ok := c.(*sqlchemy.SIntegerColumn); ok && intC.AutoIncrementOffset > 0 {
				autoInc = fmt.Sprintf(" AUTO_INCREMENT=%d", intC.AutoIncrementOffset)
			}
		}
		if c.IsIndex() {
			indexes = append(indexes, fmt.Sprintf("KEY `ix_%s_%s` (`%s`)", ts.Name(), c.Name(), c.Name()))
		}
	}
	if len(primaries) > 0 {
		cols = append(cols, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaries, ", ")))
	}
	if len(indexes) > 0 {
		cols = append(cols, indexes...)
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n%s\n) ENGINE=InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci%s", ts.Name(), strings.Join(cols, ",\n"), autoInc)
}

func (msyql *SMySQLBackend) IsSupportIndexAndContraints() bool {
	return true
}

func (mysql *SMySQLBackend) FetchTableColumnSpecs(ts sqlchemy.ITableSpec) ([]sqlchemy.IColumnSpec, error) {
	sql := fmt.Sprintf("SHOW FULL COLUMNS IN `%s`", ts.Name())
	query := ts.Database().NewRawQuery(sql, "field", "type", "collation", "null", "key", "default", "extra", "privileges", "comment")
	infos := make([]sSqlColumnInfo, 0)
	err := query.All(&infos)
	if err != nil {
		return nil, err
	}
	specs := make([]sqlchemy.IColumnSpec, 0)
	for _, info := range infos {
		specs = append(specs, info.toColumnSpec())
	}
	return specs, nil
}
