package logs

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

func NewLogger() *slog.Logger {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		location = time.FixedZone("MSK", 3*60*60)
	}

	options := &tint.Options{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key != slog.TimeKey {
				return attr
			}

			timestamp, ok := attr.Value.Any().(time.Time)
			if !ok {
				return attr
			}

			attr.Value = slog.TimeValue(timestamp.In(location))
			return attr
		},
		Level: slog.LevelDebug,
	}

	return slog.New(tint.NewHandler(os.Stderr, options))
}
