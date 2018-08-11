package hooks

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type StdioHook struct{}

func (hook *StdioHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}
	if entry.Level >= logrus.ErrorLevel {
		fmt.Fprintf(os.Stderr, line)
	} else {
		fmt.Fprintf(os.Stdout, line)
	}
	return nil
}

func (hook *StdioHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
