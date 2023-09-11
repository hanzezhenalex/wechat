package main

import (
	"github.com/hanzezhenalex/wechat/src"
	"github.com/hanzezhenalex/wechat/src/wechat"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := src.Config{
		Token:     "sdaregsghsd",
		AppID:     "wxa1e850de1191bd56",
		AppSecret: "3c87533a8b1902e37d08c5f60106bfe9",
	}

	c := wechat.NewCoordinator(cfg)

	r := gin.Default()
	r.Use(wechat.IsWechat(cfg))

	r.GET("/wechat", wechat.HealthCheck())
	r.POST("/wechat", c.Handler())

	if err := r.Run(":3000"); err != nil {
		panic(err)
	}
}
