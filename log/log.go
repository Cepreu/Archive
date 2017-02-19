// Package log replaces the built-in log interface with a wrapper for zap.
// See https://github.com/uber-go/zap for more details.
// We wrap zap to protect against compatability issues and to enable vendoring
// (since other packages can't have a direct dependency on zap); and to have
// the option to switch away from zap in the future – if need be – easily.
package log

import (
	"flag"
	"fmt"
	stdlog "log"
	"time"

	"github.com/WorkFit/commongo"
	"github.com/WorkFit/commongo/log"
	"github.com/WorkFit/commongo/testutil"
	"github.com/uber-go/zap"
)

var (
	levelFlag        = flag.String("log.level", "info", "log level: debug, info, warn, or error.")
	engine           = flag.String("log.engine", "zap", "log engine: zap (default), or human.")
	productionLogger = &zapLoggerAdapter{zap.New(zap.NewJSONEncoder(), zap.AddCaller(), zap.AddStacks(zap.ErrorLevel))}
	logger           log.LeveledLogger
)

func init() {
	flag.Parse()
	level := zap.InfoLevel
	level.UnmarshalText([]byte(*levelFlag))
	if *engine == "human" {
		logger = &testutil.Logger{InDebugMode: true}
	} else {
		productionLogger.SetLevel(level)
		logger = productionLogger
		stdlog.SetOutput(&standardLoggerAdapter{}) // redirect to the zap logger
	}
	// wire up commongo's logger with the de facto logger; setting it here means
	// that all applications that log (all of them), enable commongo's logging
	// as well – automagically
	commongo.Logger = logger
}

// CurrentLogger returns the current logger.
func CurrentLogger() log.LeveledLogger {
	return logger
}

// IsDebugEnabled checks whether or not debug logging is enabled.
// Useful to check before logging a message that requires preprocessing.
func IsDebugEnabled() bool {
	return logger.IsDebugEnabled()
}

// Debug logs a debug message.
// It accepts varargs of alternating key and value parameters.
func Debug(message string, args ...interface{}) {
	logger.Debug(message, args...)
}

// Info logs an informational message.
// It accepts varargs of alternating key and value parameters.
func Info(message string, args ...interface{}) {
	logger.Info(message, args...)
}

// Warn logs a warning message.
// It accepts varargs of alternating key and value parameters.
func Warn(message string, args ...interface{}) {
	logger.Warn(message, args...)
}

// Error logs an error message.
// It accepts varargs of alternating key and value parameters.
func Error(message string, args ...interface{}) {
	logger.Error(message, args...)
}

// ErrorObject logs an error object.
func ErrorObject(err error) {
	logger.Error(err.Error())
}

type zapLoggerAdapter struct {
	zap.Logger
}

// IsDebugEnabled checks whether or not debug logging is enabled.
// Useful to check before logging a message that requires preprocessing.
func (adapter *zapLoggerAdapter) IsDebugEnabled() bool {
	return adapter.Logger.Level() == zap.DebugLevel
}

// Debug logs a debug message.
// It accepts varargs of alternating key and value parameters.
func (adapter *zapLoggerAdapter) Debug(message string, args ...interface{}) {
	adapter.Logger.Debug(message, convertLogParameters(args)...)
}

// Info logs an informational message.
// It accepts varargs of alternating key and value parameters.
func (adapter *zapLoggerAdapter) Info(message string, args ...interface{}) {
	adapter.Logger.Info(message, convertLogParameters(args)...)
}

// Warn logs a warning message.
// It accepts varargs of alternating key and value parameters.
func (adapter *zapLoggerAdapter) Warn(message string, args ...interface{}) {
	adapter.Logger.Warn(message, convertLogParameters(args)...)
}

// Error logs an error message.
// It accepts varargs of alternating key and value parameters.
func (adapter *zapLoggerAdapter) Error(message string, args ...interface{}) {
	adapter.Logger.Error(message, convertLogParameters(args)...)
}

// convertLogParameters converts generic key value pairs into zap fields.
func convertLogParameters(args []interface{}) []zap.Field {
	fields := []zap.Field{}
	for i := 0; i < len(args); i += 2 {
		var field zap.Field
		key := args[i].(string)
		value := args[i+1]
		switch converted := value.(type) {
		case bool:
			field = zap.Bool(key, converted)
		case int:
			field = zap.Int(key, converted)
		case int64:
			field = zap.Int64(key, converted)
		case float64:
			field = zap.Float64(key, converted)
		case string:
			field = zap.String(key, converted)
		case fmt.Stringer:
			field = zap.Stringer(key, converted)
		case time.Duration:
			field = zap.Duration(key, converted)
		case time.Time:
			field = zap.Time(key, converted)
		case error:
			field = zap.String(key, converted.Error())
		default:
			field = zap.Object(key, converted) // a bit slower
		}
		fields = append(fields, field)
	}
	return fields
}

type standardLoggerAdapter struct{}

const prefixLength = len("2017/07/07 07:07:07 ")

func (*standardLoggerAdapter) Write(p []byte) (int, error) {
	logger.Error(string(p[prefixLength:])) // skip timestamp prefix
	return len(p), nil
}

// EnterTestMode readies the logger for unit testing.
func EnterTestMode() {
	adapter := &zapLoggerAdapter{zap.New(zap.NewJSONEncoder(zap.NoTime()))}
	adapter.SetLevel(zap.DebugLevel)
	logger = adapter
}

// ExitTestMode restores settings for logging in production.
func ExitTestMode() {
	logger = productionLogger
}
