package log

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/util"
	"os"
	"time"
)

const ScopeFieldName = "scope"

var Logger zerolog.Logger

func InitLogger(cfg *util.Config) {
	partsOrder := []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		ScopeFieldName,
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
		FieldsExclude: []string{ScopeFieldName},
	}

	Logger = zerolog.New(consoleWriter)
	if *cfg.Debug {
		Logger = Logger.Level(zerolog.DebugLevel)
	} else {
		Logger = Logger.Level(zerolog.InfoLevel)
	}
	Logger = Logger.With().Timestamp().Logger()
}

func formatScopeValue(vs map[string]any) {
	if scope, ok := vs[ScopeFieldName].(string); ok {
		vs[ScopeFieldName] = fmt.Sprintf("[%s]", scope)
	} else {
		vs[ScopeFieldName] = ""
	}
}
