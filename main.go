package main

import (
	"context"
	"os"

	"github.com/hanzezhenalex/wechat/src"
	"github.com/hanzezhenalex/wechat/src/datastore"
	"github.com/hanzezhenalex/wechat/src/wechat"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := src.Config{
		Token:     "sdaregsghsd",
		AppID:     "wxa1e850de1191bd56",
		AppSecret: "3c87533a8b1902e37d08c5f60106bfe9",
	}

	store, err := datastore.NewMysqlDatastore(context.Background(), cfg, true)
	if err != nil {
		logrus.Errorf("fail to create mysql datastore, err=%s", err.Error())
		os.Exit(1)
	}

	c, err := wechat.NewCoordinator(cfg, store)
	if err != nil {
		logrus.Errorf("fail to create coordinator, err=%s", err.Error())
		os.Exit(1)
	}

	r := gin.Default()
	r.Use(wechat.IsWechat(cfg))

	r.GET("/wechat", wechat.HealthCheck())
	r.POST("/wechat", c.Handler())

	c.RegisterEndpoints(r.Group("/api/v1"))

	if err := r.Run(":3000"); err != nil {
		panic(err)
	}
}
