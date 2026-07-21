package logger

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type contextKey string

const TraceIdKey contextKey = "trace_id"

type Logger struct {
	log     zerolog.Logger
	service string
}

func NewLogger(service string) *Logger {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &Logger{log: log, service: service}
}

func (l *Logger) Log(message string, data map[string]any) {
	event := l.log.
		Info().
		Str("service", l.service)

	if data != nil {
		event = event.Interface("data", data)
	}

	event.Msg(message)
}

func (l *Logger) Info(traceId string, eventName string, message string, data map[string]any) {
	event := l.log.Info().
		Str("trace_id", traceId).
		Str("service", l.service).
		Str("eventName", eventName)

	if data != nil {
		event = event.Interface("data", data)
	}

	event.Msg(message)
}

func (l *Logger) Error(traceId string, eventName string, message string, err error, data map[string]any) {
	event := l.log.Error().
		Str("trace_id", traceId).
		Str("service", l.service).
		Str("eventName", eventName).
		Err(err)

	if data != nil {
		event = event.Interface("data", data)
	}

	event.Msg(message)
}

func (l *Logger) Warn(traceId string, eventName string, message string, data map[string]any) {

	event := l.log.Warn().
		Str("trace_id", traceId).
		Str("service", l.service).
		Str("eventName", eventName)

	if data != nil {
		event = event.Interface("data", data)
	}

	event.Msg(message)
}

func (l *Logger) Debug(traceId string, eventName string, message string) {
	l.log.Debug().
		Str("trace_id", traceId).
		Str("service_name", l.service).
		Str("eventName", eventName).
		Msg(message)
}

func (l *Logger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceId := r.Header.Get("X-Trace-Id")
		if traceId == "" {
			traceId = fmt.Sprintf("%d", time.Now().UnixNano())
		}

		// Store trace ID in context
		ctx := context.WithValue(r.Context(), TraceIdKey, traceId)
		r = r.WithContext(ctx)

		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		l.log.Info().
			Str("trace_id", traceId).
			Str("service", l.service).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapped.status).
			Dur("duration", time.Since(start)).
			Msg("request completed")
	})
}

func TraceIdFromContext(ctx context.Context) string {
	if traceId, ok := ctx.Value(TraceIdKey).(string); ok {
		return traceId
	}
	return ""
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
