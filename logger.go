package photofunia

// Field represents a key-value pair used for structured logging.
// It allows attaching additional context to log messages.
type Field struct {
	Key   string
	Value interface{}
}

// Logger defines the interface for logging within the photofunia package.
// Implementations of this interface can be provided to customize logging behavior.
type Logger interface {
	// Debug logs a message at debug level with optional fields for additional context.
	Debug(msg string, fields ...Field)

	// Info logs a message at info level with optional fields for additional context.
	Info(msg string, fields ...Field)
}

// NoopLogger is a logger implementation that discards all log messages.
// It's used as the default logger when none is provided.
type NoopLogger struct{}

// Debug implements the Logger interface but does nothing with the log message.
func (l NoopLogger) Debug(msg string, fields ...Field) {}

// Info implements the Logger interface but does nothing with the log message.
func (l NoopLogger) Info(msg string, fields ...Field) {}
