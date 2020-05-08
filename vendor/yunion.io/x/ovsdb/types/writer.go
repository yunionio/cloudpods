package types

import (
	"fmt"
	"io"
)

type writer struct {
	w io.Writer
}

func (w *writer) Writef(f string, as ...interface{}) {
	b := fmt.Sprintf(f, as...)
	w.w.Write([]byte(b))
	w.w.Write([]byte("\n"))
}

func (w *writer) Writeln(ln string) {
	w.w.Write([]byte(ln))
	w.w.Write([]byte("\n"))
}
