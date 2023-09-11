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
	tm *tokenManager
}

func NewCoordinator(cfg src.Config) *coordinator {
	return &coordinator{
		tm: NewTokenManager(cfg),
	}
}

func (c *coordinator) Handler() gin.HandlerFunc {
	return func(context *gin.Context) {
		var msg message
		if err := xml.NewDecoder(context.Request.Body).Decode(&msg); err != nil {
			cTracer.Errorf("fail to decode request body, %s", err.Error())
			context.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = context.Request.Body.Close()

		_, _ = context.Writer.WriteString("")
	}
}

// use struct + inline type?
type message map[string]interface{}

func (m message) fromUsername() string {
	return m["FromUsername"].(string)
}

func (m message) msgType() string {
	return m["MsgType"].(string)
}
