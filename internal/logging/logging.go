// Package logging builds the server's two loggers: the MeshKit logger the
// server itself uses, and the slog logger the MCP SDK takes. Both write the
// same fields (time, level, msg, app) in the same format, so a log collector
// sees one shape whichever logger wrote a line.
package logging

import (
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/meshery/meshkit/logger"
	"github.com/sirupsen/logrus"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
)

// New returns the MeshKit logger and the SDK logger for format and level,
// both writing to w. Over stdio, w must not be stdout, which carries the
// MCP protocol.
func New(app, format, level string, w io.Writer) (logger.Handler, *slog.Logger, error) {
	meshkitFormat := logger.Format(logger.SyslogLogFormat)
	if format == config.LogFormatJSON {
		meshkitFormat = logger.JsonLogFormat
	}
	mk, err := logger.New(app, logger.Options{
		Format:   meshkitFormat,
		LogLevel: int(logrusLevel(level)),
		// MeshKit writes to stdout unless told otherwise.
		Output: w,
	})
	if err != nil {
		return nil, nil, err
	}

	opts := &slog.HandlerOptions{Level: slogLevel(level), ReplaceAttr: likeMeshKit}
	var h slog.Handler = slog.NewTextHandler(w, opts)
	if format == config.LogFormatJSON {
		h = slog.NewJSONHandler(w, opts)
	}
	return mk, slog.New(h).With("app", app), nil
}

// likeMeshKit writes slog's time and level the way MeshKit's logrus
// formatters do: RFC 3339 time to the second, and lowercase level names
// with "warning" for warn.
func likeMeshKit(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		if t, ok := a.Value.Any().(time.Time); ok {
			return slog.String(slog.TimeKey, t.Format(time.RFC3339))
		}
	case slog.LevelKey:
		if l, ok := a.Value.Any().(slog.Level); ok {
			return slog.String(slog.LevelKey, levelName(l))
		}
	}
	return a
}

func levelName(l slog.Level) string {
	switch l {
	case slog.LevelDebug:
		return "debug"
	case slog.LevelInfo:
		return "info"
	case slog.LevelWarn:
		return "warning"
	case slog.LevelError:
		return "error"
	default:
		return strings.ToLower(l.String())
	}
}

func logrusLevel(level string) logrus.Level {
	switch level {
	case config.LogLevelDebug:
		return logrus.DebugLevel
	case config.LogLevelWarn:
		return logrus.WarnLevel
	case config.LogLevelError:
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}

func slogLevel(level string) slog.Level {
	switch level {
	case config.LogLevelDebug:
		return slog.LevelDebug
	case config.LogLevelWarn:
		return slog.LevelWarn
	case config.LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
