package log

import (
	"fmt"
	"io"
	"os"
)

var (
	DefaultHandler = StreamHandler{
		W:   os.Stderr,
		Fmt: twoLineFormatter,
	}
	Default        Logger // Inited after GO_LOG is parsed.
	DiscardHandler = StreamHandler{
		W:   io.Discard,
		Fmt: func(Record) []byte { return nil },
	}
)

// Logs a message to the [Default] Logger with text formatted in the style of [fmt.Printf], with the
// given [Level].
func Levelf(level Level, format string, a ...interface{}) {
	Default.LazyLog(level, func() Msg {
		return Fmsg(format, a...).Skip(1)
	})
}

// Prints the arguments to the [Default] Logger in the style of [fmt] functions of similar names.
func Printf(format string, a ...interface{}) {
	Default.Log(Fmsg(format, a...).Skip(1))
}

// Prints the arguments to the [Default] Logger in the style of [fmt] functions of similar names.
func Print(a ...interface{}) {
	// TODO: There's no "Print" equivalent constructor for a Msg, and I don't know what I'd call it.
	Str(fmt.Sprint(a...)).Skip(1).Log(Default)
}

// Prints the arguments to the [Default] Logger in the style of [fmt] functions of similar names.
func Println(a ...interface{}) {
	Default.LazyLogDefaultLevel(func() Msg {
		return Msgln(a...).Skip(1)
	})
}
