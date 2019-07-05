// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prettytable

import (
	"bytes"
	"strings"
	"unicode"
)

type AlignmentType uint8

const (
	AlignLeft AlignmentType = iota
	AlignCenter
	AlignRight
)

const TabWidth = 8

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

func textCell(buf *bytes.Buffer, col ptColumn, width int, nthCol int, prevWidth int) {
	var padLeft, padRight int
	celWidth := cellDisplayWidth(col.Title, nthCol, prevWidth)
	switch col.Align {
	case AlignLeft:
		padLeft = 0
		padRight = width - celWidth
	case AlignRight:
		padLeft = width - celWidth
		padRight = 0
	default: //case AlignCenter:
		padLeft = (width - celWidth) / 2
		padRight = width - padLeft - celWidth
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
		widthAcc := 0
		for i, w := range widths {
			buf.WriteByte(' ')
			var text string
			if j < len(splitted[i]) {
				text = splitted[i][j]
			}
			c := ptColumn{
				Title: text,
				Align: columns[i].Align,
			}
			textCell(buf, c, w, i, widthAcc)
			widthAcc += w
			buf.WriteByte(' ')
			buf.WriteByte('|')
		}
		buf.WriteByte('\n')
	}
}

func runeDisplayWidth(r rune) int {
	const puncts = "。，；：（）、？《》"
	if unicode.Is(unicode.Han, r) {
		return 2
	}
	if strings.ContainsRune(puncts, r) {
		return 2
	}
	return 1
}

// cellDisplayWidth returns display width of the cell when printed as the
// nthCol.  prevWidth is the total display width (as return by this same func)
// of previous cells in the same line
func cellDisplayWidth(cell string, nthCol int, prevWidth int) int {
	// `| ` at the front and ` | ` in the middle
	sX := prevWidth + nthCol*3 + 2
	width := 0
	lines := strings.Split(cell, "\n")
	for _, line := range lines {
		displayWidth := 0
		x := sX
		for _, c := range line {
			incr := 0
			if c != '\t' {
				incr = runeDisplayWidth(c)
			} else {
				// terminal with have the char TabWidth aligned
				incr = TabWidth - (x & (TabWidth - 1))
			}
			displayWidth += incr
			x += incr
		}
		if width < displayWidth {
			width = displayWidth
		}
	}
	return width
}

func (this *PrettyTable) GetString(fields [][]string) string {
	if len(fields) == 0 {
		return ""
	}
	var widths = make([]int, len(this.columns))
	{
		// width of title columns
		widthAcc := 0
		for i, c := range this.columns {
			widths[i] = cellDisplayWidth(c.Title, i, widthAcc)
			widthAcc += widths[i]
		}
	}
	{
		// width of content columns
		widthAcc := 0
		for col := 0; ; col++ {
			cont := false
			colWidth := 0
			for _, line := range fields {
				if col < len(line) {
					cont = true
					cw := cellDisplayWidth(line[col], col, widthAcc)
					if colWidth < cw {
						colWidth = cw
					}
				}
			}
			if cont {
				if widths[col] < colWidth {
					widths[col] = colWidth
				}
				widthAcc += widths[col]
			} else {
				break
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
