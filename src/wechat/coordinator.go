package wechat

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/hanzezhenalex/wechat/src"
	"github.com/hanzezhenalex/wechat/src/datastore"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var cTracer = logrus.WithField("comp", "coordinator")

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
		var msg Message
		if err := xml.NewDecoder(context.Request.Body).Decode(&msg); err != nil {
			cTracer.Errorf("fail to decode request body, %s", err.Error())
			context.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = context.Request.Body.Close()

		ctx := context.Request.Context()

		if _, ok := c.ums.GetUserById(ctx, msg.FromUserName); !ok {
			_, _ = context.Writer.WriteString(msg.TextResponse(userNotRegistered))
			return
		}

		ret, err := c.svc.Handle(ctx, msg)
		if err != nil {
			cTracer.Errorf("fail to process message, %s", err.Error())
			ret = msg.TextResponse(serverInternalError)
		}
		_, _ = context.Writer.WriteString(ret)
	}
}

func (c *Coordinator) RegisterEndpoints(group *gin.RouterGroup) {
	group.POST("/ums/create", func(context *gin.Context) {
		auth := context.Request.Header.Get("x-alex-auth")
		if auth != src.DefaultApiToken {
			context.Writer.WriteHeader(http.StatusUnauthorized)
			return
		}

		var user datastore.User
		if err := json.NewDecoder(context.Request.Body).Decode(&user); err != nil {
			_, _ = context.Writer.Write([]byte(err.Error()))
			context.Writer.WriteHeader(http.StatusInternalServerError)
		}
		if err := c.ums.CreateNewUser(context.Request.Context(), user); err != nil {
			_, _ = context.Writer.Write([]byte(err.Error()))
			context.Writer.WriteHeader(http.StatusInternalServerError)
		}
		context.Writer.WriteHeader(http.StatusOK)
	})
}
