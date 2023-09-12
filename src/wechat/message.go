package wechat

import (
	"fmt"
	"time"
)

const (
	msgText  = "text"
	msgImage = "image"

	notSupportYet       = "尚不支持当前消息类型"
	serverInternalError = "服务器出现故障，请联系管理员"
	userNotRegistered   = "当前用户并为注册，不能使用本服务"

	duplicated   = "请勿重复上传"
	deduplicated = "成功"
)

type Message struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   string `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	MsgId        string `xml:"MsgId,omitempty"`
	Content      string `xml:"Content"`
	PicUrl       string `xml:"PicUrl"`
	MediaId      string `xml:"MediaId"`
	// TODO: not work, why?
	Others map[string]interface{} `xml:",innerxml"`
}

func (m Message) GetPicUrl() (string, error) {
	if m.MsgType != msgImage {
		return "", fmt.Errorf("PicUrl is only supported by image type, current type=%s", m.MsgType)
	}
	return m.PicUrl, nil
}

func (m Message) TextResponse(text string) string {
	template := "<xml>" +
		"<ToUserName><![CDATA[%s]]></ToUserName>" +
		"<FromUserName><![CDATA[%s]]></FromUserName>" +
		"<CreateTime>%d</CreateTime>" +
		"<MsgType><![CDATA[text]]></MsgType>" +
		"<Content><![CDATA[%s]]></Content>" +
		"</xml>"
	return fmt.Sprintf(template, m.FromUserName, m.ToUserName, time.Now().Unix(), text)
}
