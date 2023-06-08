package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
)

// StandardLogger struct for sentry
type StandardLogger struct {
	*logrus.Logger
}

var (
	Logger = NewLogger(true) //Logger New logger by loggerSentry and loggerLine
)

func Init() *StandardLogger {
	var baseLogger = logrus.New()
	var standard = &StandardLogger{baseLogger}
	standard.SetLevel(logrus.TraceLevel)
	standard.Formatter = &logrus.JSONFormatter{}
	return standard
}

// NewLogger New logger by  loggerLine
func NewLogger(sendToSentry bool) *StandardLogger {
	standard := Init()
	logName := os.Getenv("LOG_FILE")
	if len(logName) != 0 {
		file, err := os.OpenFile(logName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		standard.SetOutput(file)
	}
	standard.loggerLine()
	return standard
}

func UpdateLoggerLevel(Level uint32) {
	var level = logrus.Level(Level)
	if level > logrus.TraceLevel || level == 0 {
		level = logrus.TraceLevel
	}
	Logger.SetLevel(level)
}

// loggerLine for print log with line
func (logger *StandardLogger) loggerLine() {
	hookWithLine := NewContextLine()
	logger.Hooks.Add(hookWithLine)
}

func Info(ctx context.Context, from string, customize interface{}) {
	buildLogEntry(ctx, from, customize).Info()
}

func Error(ctx context.Context, from string, customize interface{}, error error) {
	buildLogEntry(ctx, from, customize, error).Error()
}

func Warn(ctx context.Context, from string, customize interface{}, errors ...error) {
	buildLogEntry(ctx, from, customize, errors...).Warn()
	return
}

func buildLogEntry(ctx context.Context, from string, customize interface{}, errors ...error) *logrus.Entry {
	fields := logrus.Fields{
		"from":   from,
		"source": customize,
	}
	if len(errors) > 0 {
		errLogs := make([]logrus.Fields, 0)
		for _, err := range errors {
			if err != nil {
				errLogs = append(errLogs, generateErrorFields(err))
			}
		}
		fields["errors"] = errLogs
	}
	return Logger.WithFields(fields)
}

func generateErrorFields(err error) logrus.Fields {
	return logrus.Fields{
		"err": err.Error(),
	}
}
