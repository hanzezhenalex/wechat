package wechat

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/hanzezhenalex/wechat/src"
	"github.com/hanzezhenalex/wechat/src/datastore"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var cTracer = func(ctx context.Context) *logrus.Entry {
	return logrus.WithField("comp", "coordinator").WithContext(ctx)
}

type Coordinator struct {
	ums *UserMngr
	svc Service
	tm  *tokenManager
}

func NewCoordinator(cfg src.Config, store datastore.DataStore) (*Coordinator, error) {
	c := &Coordinator{
		tm:  NewTokenManager(cfg),
		svc: NewDeduplication(store),
	}
	ums, err := NewUMS(store)
	if err != nil {
		return nil, fmt.Errorf("fail to create ums, %w", err)
	}
	c.ums = ums
	return c, nil
}

func (c *Coordinator) Handler() gin.HandlerFunc {
	return func(context *gin.Context) {
		ctx := context.Request.Context()
		tracer := cTracer(ctx)

		var msg Message
		if err := xml.NewDecoder(context.Request.Body).Decode(&msg); err != nil {
			tracer.Errorf("fail to decode request body, %s", err.Error())
			context.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = context.Request.Body.Close()
		tracer.Infof("new message from %s", msg.FromUserName)

		if _, ok := c.ums.GetUserById(ctx, msg.FromUserName); !ok {
			tracer.Warningf("message rejected, user %s not register", msg.FromUserName)
			_, _ = context.Writer.WriteString(msg.TextResponse(userNotRegistered))
			return
		}

		ret, err := c.svc.Handle(ctx, msg)
		if err != nil {
			tracer.Errorf("fail to process message, %s", err.Error())
			ret = msg.TextResponse(fmt.Sprintf("%s, trace_id=%s", serverInternalError, src.GetTraceId(ctx)))
		}

		tracer.Debug("message processed successfully")
		_, _ = context.Writer.WriteString(ret)
	}
}

func (c *Coordinator) RegisterEndpoints(group *gin.RouterGroup) {
	c.ums.RegisterEndpoints(group.Group("/ums"))
}
