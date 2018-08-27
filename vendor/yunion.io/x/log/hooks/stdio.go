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
		return err
	}
	fmt.Fprintf(os.Stderr, line)
	return nil
}

func (hook *StdioHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
