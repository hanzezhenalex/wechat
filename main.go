package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hanzezhenalex/wechat/src"
)

func main() {
	r := gin.Default()

	r.Use(src.GetWechatAuthHandler())

	r.GET("/", func(context *gin.Context) {
		fmt.Println(context.Request.URL.String())
	})

	if err := r.Run(":3000"); err != nil {
		panic(err)
	}
}
