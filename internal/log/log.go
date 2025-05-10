package logger

type Logger interface {
	Info(msg string, args ...any)  // Info logs an info message
	Error(msg string, args ...any) // Error logs an error message
	Debug(msg string, args ...any) // Debug logs a debug message
	Warn(msg string, args ...any)  // Warn logs a warning message
	Fatal(msg string, args ...any) // Fatal logs a fatal message and exits the program
}
