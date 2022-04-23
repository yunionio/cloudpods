package terminalparser

import (
	"log"
	"strconv"
	"strings"
)

type screenCsiFunc func(screen *Screen, params []rune)

var CSIFuncMap = map[rune]screenCsiFunc{
	'@': func(s *Screen, params []rune) {
		currentRow := s.GetCursorRow()
		switch len(params) {
		case 1:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				insertData := make([]rune, ps)
				for i := 0; i < ps; i++ {
					insertData[i] = Spaces[0]
				}
				currentRow.changeCursorToX(s.Cursor.X)
				currentRow.insertCharacters(insertData)
			}
		case 2:
			if params[len(params)-1] == Spaces[0] {
				if _, err := strconv.Atoi(string(params[0])); err == nil {
					log.Printf("Screen 不支持解析 CSI `%s` @\n", string(params))
				}
			}
		default:
			currentRow.changeCursorToX(s.Cursor.X)
			currentRow.insertCharacters([]rune{Spaces[0]})
		}
	},
	'A': func(s *Screen, params []rune) {
		/*
			CSI Ps A  Cursor Up Ps Times (default = 1) (CUU).

			CSI Ps SP A
			          Shift right Ps columns(s) (default = 1) (SR), ECMA-48.
		*/
		switch len(params) {
		case 0:
			s.Cursor.MoveUp(1)
		case 1:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.Cursor.MoveUp(ps)
			}
		case 2:
			if params[len(params)-1] == Spaces[0] {
				if ps, err := strconv.Atoi(string(params[0])); err == nil {
					s.Cursor.MoveRight(ps)
					log.Printf("Shift right %d columns(s) \n", ps)
				}
			}
		default:
			if params[len(params)-1] == Spaces[0] {
				if ps, err := strconv.Atoi(string(params[0])); err == nil {
					s.Cursor.MoveRight(ps)
					log.Printf("Shift right %d columns(s) \n", ps)

				}
			}

		}

	},
	'B': func(s *Screen, params []rune) {
		/*
			CSI Ps B  Cursor Down Ps Times (default = 1) (CUD).
		*/
		switch len(params) {
		case 0:
			s.Cursor.MoveDown(1)
		default:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.Cursor.MoveDown(ps)
			}
		}

	},
	'C': func(s *Screen, params []rune) {
		/*
			CSI Ps C  Cursor Forward Ps Times (default = 1) (CUF).
		*/
		switch len(params) {
		case 0:
			s.Cursor.MoveRight(1)
		default:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.Cursor.MoveRight(ps)
			}
		}

	},
	'D': func(s *Screen, params []rune) {
		/*
			CSI Ps D  Cursor Backward Ps Times (default = 1) (CUB).
		*/
		switch len(params) {
		case 0:
			s.Cursor.MoveLeft(1)
		default:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.Cursor.MoveLeft(ps)
			}
		}
	},

	'E': func(s *Screen, params []rune) {
		/*
			CSI Ps E  Cursor Next Line Ps Times (default = 1) (CNL).
		*/
		switch len(params) {
		case 0:
			s.Cursor.MoveDown(1)
		default:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.Cursor.MoveDown(ps)
			}
		}

	},
	'F': func(s *Screen, params []rune) {
		/*
			CSI Ps F  Cursor Preceding Line Ps Times (default = 1) (CPL).
		*/
		switch len(params) {
		case 0:
			s.Cursor.MoveUp(1)
		case 1:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.Cursor.MoveUp(ps)
			}
		default:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.Cursor.MoveUp(ps)
			}
		}
	},
	'I': func(screen *Screen, params []rune) {
		log.Println("Screen 不支持 I")
	},
	'G': func(s *Screen, params []rune) {
		switch len(params) {
		case 1:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.Cursor.X = ps
			}
		}
	},
	'H': func(s *Screen, params []rune) {
		if len(params) == 3 && params[1] == ';' {
			if row, err := strconv.Atoi(string(params[0])); err == nil {
				s.Cursor.Y = row
			}
			if column, err := strconv.Atoi(string(params[2])); err == nil {
				s.Cursor.X = column
			}
		}
		if len(params) == 0 {
			s.Cursor.MoveHome()
		}
	},
	'J': func(s *Screen, params []rune) {
		/*
			ESC J     Erase from the cursor to the end of the screen.
			CSI Ps J  Erase in Display (ED), VT100.
			            Ps = 0  ⇒  Erase Below (default).
			            Ps = 1  ⇒  Erase Above.
			            Ps = 2  ⇒  Erase All.
			            Ps = 3  ⇒  Erase Saved Lines, xterm.
		*/

		switch len(params) {
		case 0:
			s.eraseFromCursor()
		case 1:
			if ps, err := strconv.Atoi(string(params[0])); err == nil {
				switch ps {
				case 0:
					s.eraseBelow()
				case 1:
					s.eraseAbove()
				case 2:
					s.eraseAll()
				case 3:
					log.Println("screen 未处理Erase Saved Lines, xterm.")
				default:
					log.Printf("screen 未处理 Erase.%d \n", ps)
				}
			}
		case 2:
			log.Printf("screen 未处理 Erase in Display (DECSED), VT220. %s\n", string(params))
		default:
			log.Printf("screen 未处理 Erase %s\n", string(params))
		}
	},
	'K': func(s *Screen, params []rune) {

		/*
			ESC K     Erase from the cursor to the end of the line.
			CSI Ps K  Erase in Line (EL), VT100.
			            Ps = 0  ⇒  Erase to Right (default).
			            Ps = 1  ⇒  Erase to Left.
			            Ps = 2  ⇒  Erase All.

			CSI ? Ps K
			          Erase in Line (DECSEL), VT220.
			            Ps = 0  ⇒  Selective Erase to Right (default).
			            Ps = 1  ⇒  Selective Erase to Left.
			            Ps = 2  ⇒  Selective Erase All.
		*/
		switch len(params) {
		case 0:
			s.eraseEndToLine()
		default:
			paramsS := string(params)
			if strings.HasPrefix(paramsS, "?") {
				log.Printf("Screen不支持解析 CSI `%s` K\n", paramsS)
				return
			}
			if ps, err := strconv.Atoi(paramsS); err == nil {
				switch ps {
				case 0:
					s.eraseRight()
				case 1:
					s.eraseLeft()
				case 2:
					s.eraseAll()
				default:
					log.Printf("未处理erase %d\n", ps)
				}
			}

		}
	},
	'L': func(o *Screen, params []rune) {
		/*
			CSI Ps L  Insert Ps Line(s) (default = 1) (IL).
		*/
		log.Println("Screen不支持解析L")
	},
	'M': func(o *Screen, params []rune) {
		/*
			CSI Ps M  Delete Ps Line(s) (default = 1) (DL).
		*/
		log.Println("Screen不支持解析M")
	},
	'P': func(s *Screen, params []rune) {
		/*
			CSI Ps P  Delete Ps Character(s) (default = 1) (DCH).
		*/
		switch len(params) {
		case 0:
			s.deleteChars(1)
		default:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.deleteChars(ps)
			}
		}

	},
	'X': func(s *Screen, params []rune) {
		switch len(params) {
		case 0:
			s.deleteChars(1)
		default:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.deleteChars(ps)
			}
		}
	},
	'd': func(s *Screen, params []rune) {
		switch len(params) {
		case 0:
			s.Cursor.Y = 1
		default:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				s.Cursor.Y = ps
			}

		}
	},

	'l': func(screen *Screen, params []rune) {
		log.Println("Screen不支持解析l")
		screen.pasteMode = false
	},
	'h': func(screen *Screen, params []rune) {
		log.Println("Screen不支持解析h")
		screen.pasteMode = true
	},

	'm': func(s *Screen, params []rune) {
		switch len(params) {
		case 0:
		default:
			if ps, err := strconv.Atoi(string(params)); err == nil {
				switch ps {
				case 30:
					/*
						针对fish的环境特殊处理
					*/
					if s.Cursor.Y >= 1 && len(s.Rows) > 0 {
						index := s.Cursor.Y - 1
						if index >= len(s.Rows) {
							index = len(s.Rows) - 1
						}
						s.Rows[index].stopRecord()
					}

				case 90:
					if s.Cursor.Y >= 1 && len(s.Rows) > 0 {
						index := s.Cursor.Y - 1
						if index >= len(s.Rows) {
							index = len(s.Rows) - 1
						}
						s.Rows[index].startRecord()
					}
				default:
				}
			}
		}

	},
}
