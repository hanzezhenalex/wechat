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

func NewDeduplication(store datastore.DataStore) *Deduplication {
	return &Deduplication{
		store: store,
	}
}

func (dd *Deduplication) Handle(ctx context.Context, message Message) (ret string, err error) {
	defer func() {
		ret = message.TextResponse(ret)
	}()

	switch message.MsgType {
	case msgImage:
		url, err := message.GetPicUrl()
		if err != nil {
			return serverInternalError, fmt.Errorf("fail to get PicUrl, %w", err)
		}

		md5, err := getMd5FromUrl(url)
		if err != nil {
			// TODO: fallback to download pic and cal md5
			return serverInternalError, fmt.Errorf("fail to get md5, %w", err)
		}

		existed, err := dd.store.CreateRecordAndCheckHash(ctx, datastore.NewRecord(md5, message.FromUserName, url))

		switch {
		case err != nil:
			return serverInternalError, fmt.Errorf("fail to check record, %w", err)
		case !existed:
			return duplicated, nil
		default:
			return deduplicated, nil
		}
	default:
		return notSupportYet, nil
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
