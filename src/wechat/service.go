package wechat

import (
	"fmt"

	"github.com/hanzezhenalex/wechat/src/datastore"
)

type Service interface {
	Handle(message Message) (string, error)
}

type Deduplication struct {
	store datastore.DataStore
}

func (dd *Deduplication) Handle(message Message) (string, error) {
	var ret = "开发中"
	switch message.MsgType {
	case msgImage:
		url, err := message.GetPicUrl()
		if err != nil {
			return ret, fmt.Errorf("fail to get PicUrl, %w", err)
		}
		fmt.Println(url)

		return ret, nil
	default:
		return message.TextResponse(notSupportYet), nil
	}
}
