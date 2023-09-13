package src

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

var traceLogger = func(ctx context.Context) *logrus.Entry {
	return logrus.WithField("comp", "tracer").WithContext(ctx)
}

type traceIdKey struct{}

var traceLevel = []logrus.Level{
	logrus.PanicLevel,
	logrus.FatalLevel,
	logrus.ErrorLevel,
	logrus.WarnLevel,
	logrus.InfoLevel,
	logrus.DebugLevel,
	logrus.TraceLevel,
}

func init() {
	hooker := TraceHooker{}
	logrus.AddHook(hooker)
}

type TraceHooker struct{}

func (hook TraceHooker) Levels() []logrus.Level {
	return traceLevel
}
func (hook TraceHooker) Fire(entry *logrus.Entry) error {
	ctx := entry.Context

	traceId := ctx.Value(traceIdKey{})
	if traceId != nil {
		entry.Data["trace_id"] = traceId
	}
	return nil
}

func TracerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), traceIdKey{}, uuid.NewV4())
		c.Request.WithContext(ctx)

		startTime := time.Now()
		reqMethod := c.Request.Method
		reqUrl := c.Request.RequestURI

		logger := traceLogger(c.Request.Context())
		logger.Infof("[REQ] %s | %s", reqMethod, reqUrl)

		c.Next()

		endTime := time.Now()
		latencyTime := endTime.Sub(startTime)
		statusCode := c.Writer.Status()

		logger.Infof("[RESP] %d | %s | %s", statusCode, latencyTime, reqUrl)
	}
}

func GetTraceId(ctx context.Context) string {
	traceId := ctx.Value(traceIdKey{})
	if traceId != nil {
		return traceId.(string)
	}
	return ""
}
