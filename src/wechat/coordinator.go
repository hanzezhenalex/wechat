package wechat

import (
	"encoding/xml"
	"net/http"

	"github.com/hanzezhenalex/wechat/src"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var cTracer = logrus.WithField("comp", "coordinator")

type coordinator struct {
	svc Service
	tm  *tokenManager
}

func NewCoordinator(cfg src.Config) *coordinator {
	return &coordinator{
		tm:  NewTokenManager(cfg),
		svc: &Deduplication{},
	}
}

func (c *coordinator) Handler() gin.HandlerFunc {
	return func(context *gin.Context) {
		var msg Message
		if err := xml.NewDecoder(context.Request.Body).Decode(&msg); err != nil {
			cTracer.Errorf("fail to decode request body, %s", err.Error())
			context.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = context.Request.Body.Close()

		ret, err := c.svc.Handle(context.Request.Context(), msg)
		if err != nil {
			cTracer.Errorf("fail to process message, %s", err.Error())
			ret = serverInternalError
		}
		_, _ = context.Writer.WriteString(ret)
	}
}
