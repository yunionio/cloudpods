package prettytable

import (
	"bytes"
	"strings"
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
	buf.WriteByte('\n')
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
	nlines := 0
	splitted := make([][]string, len(columns))
	for i, column := range columns {
		title := strings.TrimRight(column.Title, "\n")
		s := strings.Split(title, "\n")
		splitted[i] = s
		if nlines < len(s) {
			nlines = len(s)
		}
	}
	for j := 0; j < nlines; j++ {
		buf.WriteByte('|')
		for i, w := range widths {
			buf.WriteByte(' ')
			var text string
			if j < len(splitted[i]) {
				text = splitted[i][j]
			}
			c := ptColumn{
				Align: columns[i].Align,
				Title: text,
			}
			textCell(buf, c, w)
			buf.WriteByte(' ')
			buf.WriteByte('|')
		}
		buf.WriteByte('\n')
	}
}

func cellWidth(cell string) int {
	width := 0
	lines := strings.Split(cell, "\n")
	for _, line := range lines {
		if width < len(line) {
			width = len(line)
		}
	}
	return width
}

func (this *PrettyTable) GetString(fields [][]string) string {
	if len(fields) == 0 {
		return ""
	}
	var widths = make([]int, len(this.columns))
	for i, c := range this.columns {
		widths[i] = cellWidth(c.Title)
	}
	for _, line := range fields {
		for i, cell := range line {
			cw := cellWidth(cell)
			if widths[i] < cw {
				widths[i] = cw
			}
		}
	}
	var columns = make([]ptColumn, len(this.columns))
	var buf bytes.Buffer
	rowLine(&buf, widths)
	for i := 0; i < len(columns); i++ {
		columns[i].Title = this.columns[i].Title
		columns[i].Align = AlignCenter
	}
	textLine(&buf, columns, widths)
	rowLine(&buf, widths)
	for _, line := range fields {
		for i := 0; i < len(line); i++ {
			columns[i].Title = line[i]
			columns[i].Align = this.columns[i].Align
		}
		textLine(&buf, columns, widths)
	}
	rowLine(&buf, widths)
	return buf.String()
}
