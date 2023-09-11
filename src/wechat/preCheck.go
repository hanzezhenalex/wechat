package wechat

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/hanzezhenalex/wechat/src"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var authTracer = logrus.WithField("comp", "wechat_auth")

func IsWechat(cfg src.Config) gin.HandlerFunc {
	return func(context *gin.Context) {
		signature := context.Query("signature")
		nonce := context.Query("nonce")
		timestamp := context.Query("timestamp")

		tokens := []string{cfg.Token, timestamp, nonce}

		sort.Slice(tokens, func(i, j int) bool {
			return tokens[i] < tokens[j]
		})

		hash := fmt.Sprintf("%x", sha1.Sum([]byte(strings.Join(tokens, ""))))

		if hash == signature {
			context.Next()
		} else {
			context.Writer.WriteHeader(http.StatusBadRequest)
			context.Abort()
			authTracer.Warning("illegal request, abort")
		}
	}
}

func HealthCheck() gin.HandlerFunc {
	return func(context *gin.Context) {
		echoStr := context.Query("echostr")
		if _, err := context.Writer.WriteString(echoStr); err != nil {
			authTracer.Errorf("fail to write response, err=%s", err.Error())
		}
	}
}
