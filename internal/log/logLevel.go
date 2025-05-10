package logger

// LogLevel represents the severity of log messages.
// It is used to filter log messages based on their importance.
// The levels are ordered from least to most severe:
// - Debug: Detailed information, typically of interest only when diagnosing problems.
// - Info: Confirmation that things are working as expected.
// - Warn: Indicates that something unexpected happened, or indicative of some problem in the near future.
// - Error: Indicates a more serious problem that prevented the program from performing a function.
// - Fatal: Indicates a serious error that caused the program to abort.
//
// The LogLevel type is used to define the severity of log messages.
// It is an integer type that can be compared to determine the severity of different log messages.
// The levels are ordered from least to most severe:
// - Debug: 0
// - Info: 1
// - Warn: 2
// - Error: 3
// - Fatal: 4
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)
