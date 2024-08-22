package log

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/util"
	"os"
	"time"
)

const scopeFieldName = "scope"

var logger zerolog.Logger

func GetCtxLogger(ctx context.Context) zerolog.Logger {
	return logger.With().Ctx(ctx).Logger()
}

func InitLogger(cfg *util.Config) {
	partsOrder := []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		scopeFieldName,
		zerolog.MessageFieldName,
	}

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		PartsOrder: partsOrder,
		FormatPrepare: func(m map[string]any) error {
			formatScopeValue(m)
			return nil
		},
		FieldsExclude: []string{scopeFieldName},
	}

	logger = zerolog.New(consoleWriter).Hook(scopeHook{})
	if *cfg.Debug {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}
	logger = logger.With().Timestamp().Logger()
}

func formatScopeValue(vs map[string]any) {
	if scope, ok := vs[scopeFieldName].(string); ok {
		vs[scopeFieldName] = fmt.Sprintf("[%s]", scope)
	} else {
		vs[scopeFieldName] = ""
	}
}

type scopeHook struct{}

func (h scopeHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if scope, ok := util.GetScopeFromCtx(e.GetCtx()); ok {
		e.Str(scopeFieldName, scope)
	}
}
