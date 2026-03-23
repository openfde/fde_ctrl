package logger

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// StandardLogger struct for sentry
type StandardLogger struct {
	*logrus.Logger
}

var Logger *StandardLogger
var logFile = "/var/log/fde.log"

func Logrotate() {
	// Check log file exists, if not exist, create it by logrotating
	_, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			//  create it by logrotating
			exec.Command("fde_fs", "-logrotate").Run()
		}
	}
	Logger = NewLogger() //Logger New logger by loggerSentry and loggerLine
}

func Init() *StandardLogger {
	var baseLogger = logrus.New()
	var standard = &StandardLogger{baseLogger}

	levelStr := strings.TrimSpace(os.Getenv("FDE_LOG_LEVEL"))
	if levelStr == "" {
		standard.SetLevel(logrus.ErrorLevel)
	} else {
		// try parse as level name first (e.g., "info", "warn")
		if lvl, err := logrus.ParseLevel(strings.ToLower(levelStr)); err == nil {
			standard.SetLevel(lvl)
		} else if n, err := strconv.Atoi(levelStr); err == nil {
			// fall back to numeric level
			standard.SetLevel(logrus.Level(n))
		} else {
			// invalid value, use default
			standard.SetLevel(logrus.ErrorLevel)
		}
	}

	standard.Formatter = &logrus.JSONFormatter{}
	return standard
}

// NewLogger New logger by  loggerLine
func NewLogger() *StandardLogger {
	standard := Init()
	logName := logFile
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

func Info(from string, customize interface{}) {
	buildLogEntry(from, customize).Info()
}

func Error(from string, customize interface{}, error error) {
	buildLogEntry(from, customize, error).Error()
}

func Warn(from string, customize interface{}, errors ...error) {
	buildLogEntry(from, customize, errors...).Warn()
	return
}

func buildLogEntry(from string, customize interface{}, errors ...error) *logrus.Entry {
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
