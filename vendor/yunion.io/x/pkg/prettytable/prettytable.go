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

	maxLineWidth int
	tryTermWidth bool
}

func NewPrettyTable(fields []string) *PrettyTable {
	var pt = PrettyTable{
		tryTermWidth: true,
	}
	for _, field := range fields {
		col := ptColumn{Title: field, Align: AlignLeft}
		pt.columns = append(pt.columns, col)
	}
	return &pt
}

func NewPrettyTableWithTryTermWidth(fields []string, tryTermWidth bool) *PrettyTable {
	pt := NewPrettyTable(fields)
	pt.tryTermWidth = tryTermWidth
	return pt
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

// cellDisplayWidth returns display width of the cell content (excluding bars
// and paddings) when printed as the nthCol.
//
// nthCol is 0-based
//
// prevWidth is the total display width (as return by this same func) of
// previous cells in the same line.  It will be used to calculate width of tab
// characters
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

func wrapCell(cell string, nthCol int, prevWidth int, newWidth int) string {
	sX := prevWidth + nthCol*3 + 2
	var (
		wrapped = false
		newCell = ""
		cline   = make([]rune, 0, newWidth+1)
		cw      = 0
		cx      = sX
	)
	for _, c := range cell {
		var incr int
		if c == '\n' {
			cline = append(cline, c)
			newCell += string(cline)
			cline = cline[:0]
			cw = 0
			cx = sX
			continue
		} else if c != '\t' {
			incr = runeDisplayWidth(c)
		} else {
			// terminal with have the char TabWidth aligned
			incr = TabWidth - (cx & (TabWidth - 1))
		}
		t := cw + incr
		if t < newWidth {
			cline = append(cline, c)
			cw += incr
			cx += incr
		} else if t == newWidth {
			cline = append(cline, c, '\n')
			newCell += string(cline)
			cline = cline[:0]
			cw = 0
			cx = sX
			if !wrapped {
				wrapped = true
			}
		} else {
			if len(cline) == 0 {
				newCell += string(c) + "\n"
			} else {
				newCell += string(cline) + "\n"
				cline = cline[:1]
				cline[0] = c
			}
			cw = 0
			cx = sX
			if !wrapped {
				wrapped = true
			}
		}
	}
	if len(cline) > 0 {
		newCell += string(cline)
	} else if !wrapped && newCell != "" {
		l := len(newCell) - 1
		c := newCell[l]
		if c == '\n' {
			newCell = newCell[:l]
		}
	}
	return newCell
}

func (this *PrettyTable) MaxLineWidth(w int) {
	this.maxLineWidth = w
}

func (this *PrettyTable) hasMaxLineWidth() (int, bool) {
	if this.maxLineWidth > 0 {
		return this.maxLineWidth, true
	}
	if this.tryTermWidth {
		w, err := termWidth()
		if err != nil {
			return 0, false
		}
		return w, true
	}
	return 0, false
}

func (this *PrettyTable) GetString(fields [][]string) string {
	if len(fields) == 0 {
		return ""
	}
	var (
		widths   = make([]int, len(this.columns))
		widthAcc = 0
	)
	{
		// width of title columns
		widthAcc = 0
		for i, c := range this.columns {
			widths[i] = cellDisplayWidth(c.Title, i, widthAcc)
			widthAcc += widths[i]
		}
	}
	{
		// width of content columns
		widthAcc = 0
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
	if maxLineWidth, ok := this.hasMaxLineWidth(); ok {
		nCols := len(this.columns)
		maxContWidth := maxLineWidth - 3*nCols - 1
		if maxContWidth < nCols {
			// ensure at least 1 ascii char print
			maxContWidth = nCols
		}
		if widthAcc > maxContWidth {
			assured := maxContWidth / len(this.columns)

			rem := maxContWidth
			nwrap := 0
			for _, w := range widths {
				if w <= assured {
					rem -= w
				} else {
					nwrap += 1
				}
			}
			// break long lines
			widthAcc := 0
			wrapped := rem / nwrap
			wrappedRem := rem % nwrap
			for col, w := range widths {
				if w > assured {
					newW := wrapped
					if wrappedRem > 0 {
						newW += 1
						wrappedRem -= 1
					}
					this.columns[col].Title = wrapCell(this.columns[col].Title, col, widthAcc, newW)
					for _, line := range fields {
						line[col] = wrapCell(line[col], col, widthAcc, newW)
					}
					widths[col] = newW // later tabs may need correction
					widthAcc += newW
				} else {
					widthAcc += w
				}
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
