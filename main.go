package main

import (
	"context"

	"github.com/hanzezhenalex/wechat/src"
)

func main() {
	_, err := src.NewMysqlDatastore(context.Background())
	if err != nil {
		panic(err)
	}
}
