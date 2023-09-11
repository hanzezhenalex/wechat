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
		var msg Message
		if err := xml.NewDecoder(context.Request.Body).Decode(&msg); err != nil {
			cTracer.Errorf("fail to decode request body, %s", err.Error())
			context.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = context.Request.Body.Close()

		var ret string

		switch msg.MsgType {
		case msgText:
			ret = ""
		default:
			ret = "不支持当前消息类型"
		}
		_, _ = context.Writer.WriteString(ret)
	}
}

type Message struct {
	ToUserName   string                 `xml:"ToUserName"`
	FromUserName string                 `xml:"FromUserName"`
	CreateTime   string                 `xml:"CreateTime"`
	MsgType      string                 `xml:"MsgType"`
	MsgId        string                 `xml:"MsgId,omitempty"`
	Content      string                 `xml:"Content"`
	Others       map[string]interface{} `xml:",innerxml"`
}

const (
	msgText = "text"
)
