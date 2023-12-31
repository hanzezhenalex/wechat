package wechat

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/hanzezhenalex/wechat/src/datastore"
)

var deduplicationTracer = func(ctx context.Context) *logrus.Entry {
	return logrus.WithField("comp", "deduplication").WithContext(ctx)
}

type Service interface {
	Handle(ctx context.Context, message Message) (string, error)
}

type Deduplication struct {
	store datastore.DataStore
}

func NewDeduplication(store datastore.DataStore) (*Deduplication, error) {
	dd := &Deduplication{
		store: store,
	}
	return dd, nil
}

func (dd *Deduplication) Handle(ctx context.Context, message Message) (ret string, err error) {
	defer func() {
		ret = message.TextResponse(ret)
	}()

	tracer := deduplicationTracer(ctx)
	tracer.Info("message processed by deduplication service")

	switch message.MsgType {
	case msgImage:
		url, err := message.GetPicUrl()
		if err != nil {
			return serverInternalError, fmt.Errorf("fail to get PicUrl, %w", err)
		}
		tracer.Debugf("pic url %s", url)

		md5, err := getMd5FromUrl(url)
		if err != nil {
			// TODO: fallback to download pic and cal md5
			return serverInternalError, fmt.Errorf("fail to get md5, %w", err)
		}
		tracer.Debugf("md5 %s", md5)

		existed, err := dd.exist(ctx, md5, url, message.FromUserName)

		switch {
		case err != nil:
			return serverInternalError, fmt.Errorf("fail to check record, %w", err)
		case existed:
			tracer.Info("duplicated pic")
			return duplicated, nil
		default:
			tracer.Info("inserted successfully")
			return deduplicated, nil
		}
	default:
		return notSupportYet, nil
	}
}

func (dd *Deduplication) exist(ctx context.Context, md5 string, url string, username string) (bool, error) {
	tracer := deduplicationTracer(ctx)

	record, err := datastore.NewRecordInfo(username, datastore.WaitingForConfirm, url)
	if err != nil {
		return false, fmt.Errorf("fail to create reocrd info, %w", err)
	}

	exist, err := dd.store.CreateRecord(ctx, record, md5, true)
	if err != nil {
		return false, fmt.Errorf("fail to create reocrd, %w", err)
	}
	tracer.Debugf("exsitence in store: %t", exist)

	return exist, nil
}

// https://mmbiz.qpic.cn/sz_mmbiz_jpg/JV8VqJ5QWKnUHHlLxTT4R0IhH3GpDfTFO7ePlHibCPDCxTwtCiamKW2ibdxPmNhFUKpDVtApTUSPdwTYo0Cwb02xw/0
func getMd5FromUrl(url string) (string, error) {
	tokens := strings.Split(url, "/")
	if len(tokens) == 6 {
		return tokens[4], nil
	}
	return "", fmt.Errorf("can not get md5 from url, url=%s", url)
}
