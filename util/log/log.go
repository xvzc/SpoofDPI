package log

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/util"
)

const (
	scopeFieldName   = "scope"
	traceIdFieldName = "trace_id"
)

var logger zerolog.Logger

func GetCtxLogger(ctx context.Context) zerolog.Logger {
	return logger.With().Ctx(ctx).Logger()
}

func InitLogger(cfg *util.Config) {
	partsOrder := []string{
		zerolog.LevelFieldName,
		zerolog.TimestampFieldName,
		traceIdFieldName,
		scopeFieldName,
		zerolog.MessageFieldName,
	}

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		PartsOrder: partsOrder,
		FormatPrepare: func(m map[string]any) error {
			formatFieldValue[string](m, "%s", traceIdFieldName)
			formatFieldValue[string](m, "[%s]", scopeFieldName)
			return nil
		},
		FieldsExclude: []string{traceIdFieldName, scopeFieldName},
	}

	logger = zerolog.New(consoleWriter).Hook(ctxHook{})
	if cfg.Debug {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}
	logger = logger.With().Timestamp().Logger()
}

func formatFieldValue[T any](vs map[string]any, format string, field string) {
	if v, ok := vs[field].(T); ok {
		vs[field] = fmt.Sprintf(format, v)
	} else {
		vs[field] = ""
	}
}

type ctxHook struct{}

func (h ctxHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if scope, ok := util.GetScopeFromCtx(e.GetCtx()); ok {
		e.Str(scopeFieldName, scope)
	}
	if traceId, ok := util.GetTraceIdFromCtx(e.GetCtx()); ok {
		e.Str(traceIdFieldName, traceId)
	}
}
