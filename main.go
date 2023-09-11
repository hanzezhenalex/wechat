package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/", func(context *gin.Context) {
		fmt.Println(context.Request.URL.String())
	})

	if err := r.Run(":80"); err != nil {
		panic(err)
	}
}
