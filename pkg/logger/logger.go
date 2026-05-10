package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	SessionIDKey contextKey = "session_id"
)

type Logger struct {
	log zerolog.Logger
}

func New() *Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &Logger{log: log}
}

func (l *Logger) WithRequestID(ctx context.Context) *Logger {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return &Logger{log: l.log.With().Str("request_id", reqID).Logger()}
	}
	return l
}

func (l *Logger) WithSessionID(sessionID string) *Logger {
	return &Logger{log: l.log.With().Str("session_id", sessionID).Logger()}
}

func (l *Logger) Info() *event {
	return &event{e: l.log.Info()}
}

func (l *Logger) Warn() *event {
	return &event{e: l.log.Warn()}
}

func (l *Logger) Error() *event {
	return &event{e: l.log.Error()}
}

func (l *Logger) Debug() *event {
	return &event{e: l.log.Debug()}
}

type event struct {
	e *zerolog.Event
}

func (e *event) Bool(key string, val bool) *event {
	e.e = e.e.Bool(key, val)
	return e
}

func (e *event) Int(key string, val int) *event {
	e.e = e.e.Int(key, val)
	return e
}

func (e *event) Int64(key string, val int64) *event {
	e.e = e.e.Int64(key, val)
	return e
}

func (e *event) Str(key, val string) *event {
	e.e = e.e.Str(key, val)
	return e
}

func (e *event) Dur(key string, val time.Duration) *event {
	e.e = e.e.Dur(key, val)
	return e
}

func (e *event) Err(err error) *event {
	e.e = e.e.Err(err)
	return e
}

func (e *event) Int32(key string, val int32) *event {
	e.e = e.e.Int(key, int(val))
	return e
}

func (e *event) Msg(msg string) {
	e.e.Msg(msg)
}

func (e *event) Msgf(format string, args ...interface{}) {
	e.e.Msgf(format, args...)
}