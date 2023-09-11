package src

import (
	"crypto/sha1"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

const token = "sdaregsghsd"

var authTracer = logrus.WithField("comp", "wechat_auth")

func GetWechatAuthHandler() gin.HandlerFunc {
	return func(context *gin.Context) {
		signature := context.Query("signature")
		echoStr := context.Query("echostr")
		nonce := context.Query("nonce")
		timestamp := context.Query("timestamp")

		tokens := []string{token, timestamp, nonce}

		sort.Slice(tokens, func(i, j int) bool {
			return tokens[i] < tokens[j]
		})

		hash := fmt.Sprintf("%x", sha1.Sum([]byte(strings.Join(tokens, ""))))

		if hash == signature {
			if _, err := context.Writer.WriteString(echoStr); err != nil {
				authTracer.Errorf("fail to write response, err=%s", err.Error())
			}
		} else {
			context.Writer.WriteHeader(http.StatusBadRequest)
			context.Abort()
			authTracer.Warning("fail to check auth, request abort")
		}
	}
}
