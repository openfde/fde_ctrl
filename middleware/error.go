package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"

	"fde_ctrl/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func ErrHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				debug.PrintStack()
				trace(c)
				c.Status(http.StatusInternalServerError)
				c.Abort()
				return
			}
		}()
		c.Next()
	}
}

func trace(c *gin.Context) {
	pc := make([]uintptr, 10) // at least 1 entry needed
	n := runtime.Callers(0, pc)
	stackInfoBuffer := new(bytes.Buffer)
	for i := 0; i < n; i++ {
		f := runtime.FuncForPC(pc[i])
		file, line := f.FileLine(pc[i])
		stackInfoBuffer.WriteString(fmt.Sprintf("%s:%d %s\n", file, line, f.Name()))
	}
	logSource := sourceWithCtx(c)
	logSource["stack_info"] = stackInfoBuffer.String()
	logger.Error(c.Request.Context(), "un_handled_error", logSource, nil)
}

// NewEntryWithCtx be used at pos web middle ware with context.
func sourceWithCtx(c *gin.Context) map[string]interface{} {
	var statusCode, dataLength int
	var path, clientUserAgent, referer, method, params string
	if c.Request != nil {
		params = c.Request.URL.RawQuery
		path = c.Request.URL.Path
		clientUserAgent = c.Request.UserAgent()
		referer = c.Request.Referer()
		method = c.Request.Method
	}
	if c.Writer != nil {
		statusCode = c.Writer.Status()
		dataLength = c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}
	}
	return logrus.Fields{
		"method":      method,
		"referer":     referer,
		"data_length": dataLength,
		"user_agent":  clientUserAgent,
		"path":        path,
		"http": logrus.Fields{
			"url_details": logrus.Fields{
				"path": path,
			},
			"status_code": statusCode,
			"method":      method,
			"params":      params,
		},
	}
}
