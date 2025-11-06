package log

import (
	"log/slog"
)

func toSlogLevel(level Level) (slog.Level, bool) {
	switch level {
	case Never:
		return slog.LevelDebug - 1, false
	case NotSet:
		return slog.LevelWarn - 1, false
	case Debug:
		return slog.LevelDebug, true
	case Info:
		return slog.LevelInfo, true
	case Warning:
		return slog.LevelWarn, true
	case Error:
		return slog.LevelError, true
	case Critical:
		return slog.LevelError + 1, true
	case disabled:
		return slog.LevelDebug - 1, false
	default:
		panic(level)
	}
}

func fromSlogLevel(sl slog.Level) Level {
	switch sl {
	case slog.LevelDebug:
		return Debug
	case slog.LevelInfo:
		return Info
	case slog.LevelWarn:
		return Warning
	case slog.LevelError:
		return Error
	default:
		panic(sl)
	}
}
