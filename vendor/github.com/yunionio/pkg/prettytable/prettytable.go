package prettytable

import (
	"bytes"
)

type AlignmentType uint8

const (
	AlignLeft AlignmentType = iota
	AlignCenter
	AlignRight
)

type ptColumn struct {
	Title string
	Align AlignmentType
}

type PrettyTable struct {
	columns []ptColumn
}

func NewPrettyTable(fields []string) *PrettyTable {
	var pt = PrettyTable{}
	for _, field := range fields {
		col := ptColumn{Title: field, Align: AlignLeft}
		pt.columns = append(pt.columns, col)
	}
	return &pt
}

func rowLine(buf *bytes.Buffer, widths []int) {
	buf.WriteByte('+')
	for _, w := range widths {
		buf.WriteByte('-')
		for i := 0; i < w; i++ {
			buf.WriteByte('-')
		}
		buf.WriteByte('-')
		buf.WriteByte('+')
	}
}

func textCell(buf *bytes.Buffer, col ptColumn, width int) {
	var padLeft, padRight int
	switch col.Align {
	case AlignLeft:
		padLeft = 0
		padRight = width - len(col.Title)
	case AlignRight:
		padLeft = width - len(col.Title)
		padRight = 0
	default: //case AlignCenter:
		padLeft = (width - len(col.Title)) / 2
		padRight = width - padLeft - len(col.Title)
	}
	for i := 0; i < padLeft; i++ {
		buf.WriteByte(' ')
	}
	buf.WriteString(col.Title)
	for i := 0; i < padRight; i++ {
		buf.WriteByte(' ')
	}
}

func textLine(buf *bytes.Buffer, columns []ptColumn, widths []int) {
	buf.WriteByte('|')
	for i, w := range widths {
		buf.WriteByte(' ')
		textCell(buf, columns[i], w)
		buf.WriteByte(' ')
		buf.WriteByte('|')
	}
}

func (this *PrettyTable) GetString(fields [][]string) string {
	if len(fields) == 0 {
		return ""
	}
	var widths = make([]int, len(this.columns))
	for i, c := range this.columns {
		if widths[i] < len(c.Title) {
			widths[i] = len(c.Title)
		}
	}
	for _, line := range fields {
		for i, cell := range line {
			if widths[i] < len(cell) {
				widths[i] = len(cell)
			}
		}
	}
	var columns = make([]ptColumn, len(this.columns))
	var buf bytes.Buffer
	rowLine(&buf, widths)
	buf.WriteByte('\n')
	for i := 0; i < len(columns); i++ {
		columns[i].Title = this.columns[i].Title
		columns[i].Align = AlignCenter
	}
	textLine(&buf, columns, widths)
	buf.WriteByte('\n')
	rowLine(&buf, widths)
	for _, line := range fields {
		for i := 0; i < len(line); i++ {
			columns[i].Title = line[i]
			columns[i].Align = this.columns[i].Align
		}
		buf.WriteByte('\n')
		textLine(&buf, columns, widths)
	}
	buf.WriteByte('\n')
	rowLine(&buf, widths)
	return buf.String()
}
