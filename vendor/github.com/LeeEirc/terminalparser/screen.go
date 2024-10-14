package terminalparser

import (
	"bytes"
	"log"
	"strconv"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

type Screen struct {
	Rows []*Row

	Cursor *Cursor

	pasteMode bool // Set bracketed paste mode, xterm. ?2004h   reset ?2004l

	title string
}

func (s *Screen) Parse(data []byte) []string {
	s.Cursor.Y = 1
	s.Rows = append(s.Rows, &Row{
		dataRune: make([]rune, 0, 1024),
	})
	rest := data
	for len(rest) > 0 {
		code, size := utf8.DecodeRune(rest)
		rest = rest[size:]
		switch code {
		case ESCKey:
			code, size = utf8.DecodeRune(rest)
			rest = rest[size:]
			switch code {
			case '[':
				// CSI
				rest = s.parseCSISequence(rest)
				continue
			case ']':
				// OSC
				rest = s.parseOSCSequence(rest)
				continue
			default:
				if existIndex := bytes.IndexRune([]byte(string(Intermediate)), code); existIndex >= 0 {
					// ESC
					rest = s.parseIntermediate(code, rest)
					continue
				}
				if existIndex := bytes.IndexRune([]byte(string(Parameters)), code); existIndex >= 0 {

					log.Printf("Screen 未解析 ESC `%q` %xParameters字符\n", code, code)
					continue
				}
				if existIndex := bytes.IndexRune([]byte(string(Uppercase)), code); existIndex >= 0 {
					log.Printf("Screen 未解析 ESC `%q` %x Uppercase字符\n", code, code)
					continue
				}

				if existIndex := bytes.IndexRune([]byte(string(Lowercase)), code); existIndex >= 0 {
					log.Printf("Screen 未解析 ESC `%q` %x Lowercase字符\n", code, code)
					continue
				}
				log.Printf("Screen 未解析 ESC `%q` %x\n", code, code)
			}
			continue
		case Delete:
			continue
		default:
			if existIndex := bytes.IndexRune([]byte(string(C0Control)), code); existIndex >= 0 {
				s.parseC0Sequence(code)
			} else {
				if len(s.Rows) == 0 && s.Cursor.Y == 0 {
					s.Rows = append(s.Rows, &Row{
						dataRune: make([]rune, 0, 1024),
					})
					s.Cursor.Y++
				}
				s.appendCharacter(code)
			}
			continue
		}

	}
	result := make([]string, len(s.Rows))
	for i := range s.Rows {
		result[i] = s.Rows[i].String()
	}
	return result
}

func (s *Screen) parseC0Sequence(code rune) {
	switch code {
	case 0x07:
		//bell 忽略
	case 0x08:
		// 后退1光标
		s.Cursor.MoveLeft(1)
	case 0x0d:
		/*
			\r
		*/
		s.Cursor.X = 0
		if s.Cursor.Y > len(s.Rows) {
			s.Rows = append(s.Rows, &Row{
				dataRune: make([]rune, 0, 1024),
			})
		}
	case 0x0a:
		/*
			\n
		*/
		s.Cursor.Y++
		if s.Cursor.Y > len(s.Rows) {
			s.Rows = append(s.Rows, &Row{
				dataRune: make([]rune, 0, 1024),
			})
		}
	default:
		log.Printf("未处理的字符 %q %v\n", code, code)
	}

}

func (s *Screen) parseCSISequence(p []byte) []byte {
	endIndex := bytes.IndexFunc(p, IsAlphabetic)
	params := []rune(string(p[:endIndex]))
	switch rune(p[endIndex]) {
	case 'Y':
		//	/*
		//		ESC Y Ps Ps
		//		          Move the cursor to given row and column.
		//	*/
		if len(p[endIndex+1:]) < 2 {
			return p[endIndex+1:]
		}
		if row, err := strconv.Atoi(string(p[endIndex+1])); err == nil {
			s.Cursor.Y = row
		}
		if col, err := strconv.Atoi(string(p[endIndex+2])); err == nil {
			s.Cursor.X = col
		}
		return p[endIndex+3:]

	}

	funcName, ok := CSIFuncMap[rune(p[endIndex])]
	if ok {
		funcName(s, params)
	} else {
		log.Printf("screen未处理的CSI %s %q\n", DebugString(string(params)), p[endIndex])
	}

	return p[endIndex+1:]
}

func (s *Screen) parseIntermediate(code rune, p []byte) []byte {
	switch code {
	case '(':
		terminationIndex := bytes.IndexFunc(p, func(r rune) bool {
			if insideIndex := bytes.IndexRune([]byte(string(Alphabetic)), r); insideIndex < 0 {
				return false
			}
			return true
		})
		params := p[:terminationIndex+1]
		switch string(params) {
		case "B":
			/*
				ESC ( C   Designate G0 Character Set, VT100, ISO 2022.
						  C = B  ⇒  United States (USASCII), VT100.
			*/
		}
		p = p[terminationIndex+1:]
		return p
	case ')':
		terminationIndex := bytes.IndexFunc(p, func(r rune) bool {
			if insideIndex := bytes.IndexRune([]byte(string(Alphabetic)), r); insideIndex < 0 {
				return false
			}
			return true
		})
		p = p[terminationIndex+1:]
	default:
		log.Printf("Screen 未解析 ESC `%q` %x Intermediate字符\n", code, code)
	}
	return p
}

func (s *Screen) parseOSCSequence(p []byte) []byte {
	if endIndex := bytes.IndexRune(p, BEL); endIndex >= 0 {
		return p[endIndex+1:]
	}

	if endIndex := bytes.IndexRune(p, ST); endIndex >= 0 {
		return p[endIndex+1:]
	}
	log.Println("未处理的 parseOSCSequence")
	return p
}

func (s *Screen) appendCharacter(code rune) {
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.appendCharacter(code)
	width := runewidth.StringWidth(string(code))
	s.Cursor.X += width
}

func (s *Screen) eraseEndToLine() {
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.eraseRight()

}

func (s *Screen) eraseRight() {
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.eraseRight()
}

func (s *Screen) eraseLeft() {
	log.Printf("Screen %s Erase Left cursor(%d，%d) 总Row数量 %d",
		UnsupportedMsg, s.Cursor.X, s.Cursor.Y, len(s.Rows))
}

func (s *Screen) eraseAbove() {
	s.Rows = s.Rows[s.Cursor.Y-1:]
}

func (s *Screen) eraseBelow() {
	s.Rows = s.Rows[:s.Cursor.Y]
}

func (s *Screen) eraseAll() {
	s.Rows = s.Rows[:0]
	//htop?
	s.Cursor.X = 0
	s.Cursor.Y = 0
}

func (s *Screen) eraseFromCursor() {
	if s.Cursor.Y > len(s.Rows) {
		s.Cursor.Y = len(s.Rows)
	}
	s.Rows = s.Rows[:s.Cursor.Y]
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.eraseRight()
}

func (s *Screen) deleteChars(ps int) {
	currentRow := s.GetCursorRow()
	currentRow.changeCursorToX(s.Cursor.X)
	currentRow.deleteChars(ps)
}

func (s *Screen) GetCursorRow() *Row {
	if s.Cursor.Y == 0 {
		s.Cursor.Y++
	}
	if len(s.Rows) == 0 {
		s.Rows = append(s.Rows, &Row{
			dataRune: make([]rune, 0, 1024),
		})
	}
	index := s.Cursor.Y - 1
	if index >= len(s.Rows) {
		log.Printf("总行数 %d 比当前行 %d 小，可能存在解析错误 \n", len(s.Rows), s.Cursor.Y)
		return s.Rows[len(s.Rows)-1]
	}
	return s.Rows[s.Cursor.Y-1]
}

const UnsupportedMsg = "Unsupported"
