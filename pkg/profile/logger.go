package profile

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	// Style definitions
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)

	debugStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
)

type DefaultLogger struct {
	logger *log.Logger
	level  LogLevel
	writer io.Writer
	quiet  bool
}

// SetQuiet suppresses Success messages (the context-switch confirmations)
// when q is true. It returns the logger so it can be chained onto NewLogger.
func (l *DefaultLogger) SetQuiet(q bool) *DefaultLogger {
	l.quiet = q
	return l
}

func NewLogger(level string) *DefaultLogger {
	parsed := parseLevel(level)
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Prefix:          "azctx",
		// Without this the logger defaults to InfoLevel and drops every
		// Debug record, whatever --log-level says.
		Level: logLevels[parsed],
		// Skip this package's wrapper methods so the caller is the code
		// that logged, not logger.go.
		CallerOffset: 1,
	})

	return &DefaultLogger{
		logger: logger,
		level:  parsed,
		writer: os.Stderr,
	}
}

var logLevels = map[LogLevel]log.Level{
	LevelDebug: log.DebugLevel,
	LevelInfo:  log.InfoLevel,
	LevelWarn:  log.WarnLevel,
	LevelError: log.ErrorLevel,
}

func parseLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l *DefaultLogger) formatMessage(msg string, args ...interface{}) string {
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	if l.level <= LevelDebug {
		formattedMsg := l.formatMessage(msg, args...)
		l.logger.Debug(debugStyle.Render(formattedMsg))
	}
}

func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	if l.level <= LevelInfo {
		formattedMsg := l.formatMessage(msg, args...)
		// For Info, we'll use a simpler, user-friendly output
		fmt.Println(infoStyle.Render(formattedMsg))
	}
}

func (l *DefaultLogger) Success(msg string, args ...interface{}) {
	if l.quiet {
		return
	}
	if l.level <= LevelInfo {
		formattedMsg := l.formatMessage(msg, args...)
		fmt.Println(successStyle.Render(formattedMsg))
	}
}

func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	if l.level <= LevelWarn {
		formattedMsg := l.formatMessage(msg, args...)
		l.logger.Warn(warnStyle.Render(formattedMsg))
	}
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	if l.level <= LevelError {
		formattedMsg := l.formatMessage(msg, args...)
		l.logger.Error(errorStyle.Render(formattedMsg))
	}
}
