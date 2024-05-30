package logger

import (
	"fmt"
	"io"
	"log"
)

type Logger struct {
	onLog  func(format string, a ...any)
	logger *log.Logger
	prefix string
}

func NewLogger() *Logger {
	logger := log.New(io.Discard, "", log.Default().Flags())

	return &Logger{
		onLog:  func(format string, a ...any) {},
		logger: logger,
		prefix: "",
	}
}

func (l *Logger) SetPrefix(prefix string) {
	l.logger.SetPrefix(prefix)
	l.prefix = fmt.Sprintf("%s: ", prefix)
}

func (l *Logger) GetPrefix() string {
	return l.logger.Prefix()
}

func (l *Logger) Writer() io.Writer {
	return l.logger.Writer()
}

func (l *Logger) SetOutput(writer io.Writer) {
	l.logger.SetOutput(writer)
}

func (l *Logger) SetOnLog(hook func(format string, a ...any)) {
	l.onLog = hook
}

func (l *Logger) Log(format string, a ...any) {
	newFmt := fmt.Sprintf("%s%s", l.prefix, format)
	if l.onLog != nil {
		l.onLog(newFmt, a...)
	}
	newFmt += "\n"
	l.logger.Printf(newFmt, a...)
}
