package log

import (
	"os"
)

func init() {
	var err error
	rules, err = parseEnvRules()
	if err != nil {
		panic(err)
	}
	// The default formatter should be parsed here when it is implemented.
	Default = loggerCore{
		nonZero: true,
		// This is the level if no rules apply, unless overridden in this logger, or any derived
		// loggers.
		filterLevel: Warning,
		Handlers:    []Handler{DefaultHandler},
	}.asLogger()
	Default.defaultLevel, _, err = levelFromString(os.Getenv(EnvDefaultLevel))
	if err != nil {
		panic(err)
	}
}
