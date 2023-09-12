package wechat

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXmlDecode(t *testing.T) {
	rq := require.New(t)
	msg := "<xml>" +
		"<ToUserName><![CDATA[toUser]]></ToUserName>" +
		"<FromUserName><![CDATA[fromUser]]></FromUserName>" +
		"<CreateTime>1348831860</CreateTime>" +
		"<MsgType><![CDATA[text]]></MsgType>" +
		"<Content><![CDATA[this is a test]]></Content>" +
		"<MsgId>1234567890123456</MsgId>" +
		"<MsgDataId>xxxx</MsgDataId>" +
		"<Idx>xxxx</Idx>" +
		"</xml>"

	buf := bytes.NewBuffer([]byte(msg))
	var decoded Message

	rq.NoError(xml.NewDecoder(buf).Decode(&decoded))

	rq.Equal(decoded.ToUserName, "toUser")
	rq.Equal(decoded.FromUserName, "fromUser")
	rq.Equal(decoded.MsgType, "text")
	rq.Equal(decoded.Content, "this is a test")
}
