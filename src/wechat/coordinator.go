package wechat

import (
	"fmt"
	"io/ioutil"
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
		raw, err := ioutil.ReadAll(context.Request.Body)
		if err != nil {
			cTracer.Errorf("fail to read request body, %s", err.Error())
			context.Writer.WriteHeader(http.StatusBadRequest)
		}
		_ = context.Request.Body.Close()

		fmt.Println(string(raw))
		_, _ = context.Writer.WriteString("")
	}
}
