package wechat

import (
	"context"
	"fmt"
	"strings"

	"github.com/hanzezhenalex/wechat/src/datastore"
)

type Service interface {
	Handle(ctx context.Context, message Message) (string, error)
}

type Deduplication struct {
	store datastore.DataStore
}

func (dd *Deduplication) Handle(_ context.Context, message Message) (string, error) {
	switch message.MsgType {
	case msgImage:
		url, err := message.GetPicUrl()
		if err != nil {
			return serverInternalError, fmt.Errorf("fail to get PicUrl, %w", err)
		}
		md5, err := getMd5FromUrl(url)
		if err != nil {
			return serverInternalError, fmt.Errorf("fail to get PicUrl, %w", err)
		}
		return md5, nil
	default:
		return message.TextResponse(notSupportYet), nil
	}
}

// https://mmbiz.qpic.cn/sz_mmbiz_jpg/JV8VqJ5QWKnUHHlLxTT4R0IhH3GpDfTFO7ePlHibCPDCxTwtCiamKW2ibdxPmNhFUKpDVtApTUSPdwTYo0Cwb02xw/0
func getMd5FromUrl(url string) (string, error) {
	tokens := strings.Split(url, "/")
	if len(tokens) == 6 {
		return tokens[4], nil
	}
	return "", fmt.Errorf("can not get md5 from url, url=%s", url)
}
