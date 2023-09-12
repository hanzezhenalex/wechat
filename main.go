package main

import (
	"context"
	"flag"
	"os"

	"github.com/hanzezhenalex/wechat/src"
	"github.com/hanzezhenalex/wechat/src/datastore"
	"github.com/hanzezhenalex/wechat/src/wechat"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const defaultConfigFilePath = "./config.json"

var configFilePath string

func init() {
	flag.StringVar(&configFilePath, "config", defaultConfigFilePath, "config file path")

	flag.Parse()
}

func main() {
	cfg, err := src.NewConfigFromFile(configFilePath)
	if err != nil {
		logrus.Errorf("fail to read config, err=%s", err.Error())
		os.Exit(1)
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

	wechatGroup := r.Group("/wechat")
	wechatGroup.Use(wechat.IsWechat(cfg))
	wechatGroup.GET("/portal", wechat.HealthCheck())
	wechatGroup.POST("/portal", c.Handler())

	c.RegisterEndpoints(r.Group("/internal/api/v1"))

	if err := r.Run(":3000"); err != nil {
		panic(err)
	}
}
