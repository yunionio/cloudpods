package cli_util

import (
	"encoding/json"
	"fmt"

	"yunion.io/x/ovsdb/types"
)

type List struct {
	Headings ListHeadings
	Data     []ListDataRow
}
type ListHeadings []string
type ListDataRow []ListDataColumn
type ListDataColumn = interface{}

func (h ListHeadings) getNth(i int) (string, error) {
	if i < len(h) {
		return h[i], nil
	}
	return "", fmt.Errorf("bad index into list headings (%d>=%d)", i, len(h))
}

func UnmarshalJSON(data []byte, irows types.ITable) error {
	list := &List{}
	if err := json.Unmarshal(data, list); err != nil {
		return err
	}
	for _, row := range list.Data {
		irow := irows.NewRow()
		for colI := range row {
			col := row[colI]
			colName, err := list.Headings.getNth(colI)
			if err != nil {
				return err
			}
			if err := irow.SetColumn(colName, col); err != nil {
				return err
			}
		}
		irows.AppendRow(irow)
	}
	return nil
}
