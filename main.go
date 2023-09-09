package main

import (
	"context"

	"github.com/hanzezhenalex/wechat/src"
)

func main() {
	_, _ = src.NewMysqlDatastore(context.Background())
}
