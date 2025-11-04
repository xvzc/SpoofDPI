package applog

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/appctx"
)

const (
	// scopeFieldName defines the key for the "scope" field in structured logs.
	scopeFieldName = "scope"
	// traceIDFieldName defines the key for the "trace_id" field in structured logs.
	traceIDFieldName = "trace_id"
)

// NewLogger creates and configures a new zerolog.Logger instance
// based on the application configuration.
// This instance is intended to be passed to other components via Dependency Injection.
func NewLogger(debug bool) zerolog.Logger {
	// Define the order of parts in the console output.
	partsOrder := []string{
		zerolog.LevelFieldName,
		zerolog.TimestampFieldName,
		traceIDFieldName, // Custom fields are placed before the message.
		scopeFieldName,
		zerolog.MessageFieldName,
	}

	// Configure a human-readable console writer.
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		PartsOrder: partsOrder,
		// FormatPrepare intercepts fields just before printing
		// to apply custom formatting, like adding brackets [SCOPE] or parens (trace_id).
		FormatPrepare: func(m map[string]any) error {
			if v, ok := m[traceIDFieldName].(string); ok && v != "" {
				m[traceIDFieldName] = v
			} else {
				// If trace_id is not present or is empty, ensure the value is an empty string
				// to prevent zerolog from printing a nil-related error.
				m[traceIDFieldName] = ""
			}

			if v, ok := m[scopeFieldName].(string); ok && v != "" {
				m[scopeFieldName] = fmt.Sprintf("[%s]", v)
			} else {
				// Do the same for scope for consistency.
				m[scopeFieldName] = ""
			}
			return nil
		},
		// Exclude the raw field names since we have already formatted them
		// in FormatPrepare. This prevents duplicate output (e.g., [SCOPE] scope="SCOPE").
		FieldsExclude: []string{traceIDFieldName, scopeFieldName},
	}

	// Create the base logger instance with the console writer and attach the hook.
	logger := zerolog.New(consoleWriter).Hook(ctxHook{})

	// Set the appropriate log level based on the configuration.
	if debug {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	// Add a timestamp to all log events and return the final logger.
	return logger.With().Timestamp().Logger()
}

// WithScope is a helper for components (like HttpHandler or DnsResolver)
// to create a sub-logger with their component name.
func WithScope(logger zerolog.Logger, scope string) zerolog.Logger {
	return logger.With().Str(scopeFieldName, scope).Logger()
}

// ctxHook implements the zerolog.Hook interface.
// Its Run method is called for every log event, allowing us to
// automatically extract values from the context.
type ctxHook struct{}

// Run extracts request-scoped values (trace_id) from the context
// and adds them to the log event automatically.
// This hook is triggered only if .Ctx(ctx) is added to the log chain.
func (h ctxHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Check if a context was attached to the event using .Ctx(ctx).
	ctx := e.GetCtx()
	if ctx == nil {
		return
	}

	// Request-scoped values like trace_id.
	// Scope is expected to be added at the component's creation time.
	if traceId, ok := appctx.TraceIDFrom(ctx); ok {
		e.Str(traceIDFieldName, traceId)
	}
}
