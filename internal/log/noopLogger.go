package logger

type NoOpLogger struct{}

func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

func (l *NoOpLogger) Info(msg string, args ...any)  {}
func (l *NoOpLogger) Error(msg string, args ...any) {}
func (l *NoOpLogger) Debug(msg string, args ...any) {}
func (l *NoOpLogger) Warn(msg string, args ...any)  {}
func (l *NoOpLogger) Fatal(msg string, args ...any) {}
