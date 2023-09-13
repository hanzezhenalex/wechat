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

const (
	defaultConfigFilePath = "./config.json"

	internalV1Group = "/internal/api/v1"
	wechatGroup     = "/wechat"
	portal          = "/portal"
)

var (
	configFilePath string
	debug          bool
)

func init() {
	flag.StringVar(&configFilePath, "config", defaultConfigFilePath, "config file path")
	flag.BoolVar(&debug, "debug", false, "debug mode")

	flag.Parse()
}

func main() {
	cfg, err := src.NewConfigFromFile(configFilePath)
	if err != nil {
		logrus.Errorf("fail to read config, err=%s", err.Error())
		os.Exit(1)
	}

	store, err := datastore.NewMysqlDatastore(context.Background(), cfg, false)
	if err != nil {
		logrus.Errorf("fail to create mysql datastore, err=%s", err.Error())
		os.Exit(1)
	}

	c, err := wechat.NewCoordinator(cfg, store)
	if err != nil {
		logrus.Errorf("fail to create coordinator, err=%s", err.Error())
		os.Exit(1)
	}


	if err := startGin(cfg, c); err != nil {
		logrus.Errorf("fail to run gin server, err=%s", err.Error())
		os.Exit(1)
	}

}

func startGin(cfg src.Config, c *wechat.Coordinator) error {
	eng := gin.New()
	eng.Use(gin.Recovery(), src.TracerMiddleware())

	if debug == false {
		gin.SetMode(gin.ReleaseMode)
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		gin.SetMode(gin.DebugMode)
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	registerRoutes(eng, c, cfg)

	return eng.Run(":8096")
}

func registerRoutes(r *gin.Engine, c *wechat.Coordinator, cfg src.Config) {
	wechatG := r.Group(wechatGroup)
	wechatG.Use(wechat.IsWechat(cfg))
	wechatG.GET(portal, wechat.HealthCheck())
	wechatG.POST(portal, c.Handler())

	c.RegisterEndpoints(r.Group(internalV1Group))
}
