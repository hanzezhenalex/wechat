package wechat

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/hanzezhenalex/wechat/src"
)

type tokenManager struct {
	token atomic.Value // string

	client *http.Client
	url    string
}

func NewTokenManager(cfg src.Config) *tokenManager {
	tm := &tokenManager{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		url: fmt.Sprintf(
			"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
			cfg.AppID, cfg.AppSecret,
		),
	}
	go tm.daemon()
	return tm
}

func (tm *tokenManager) daemon() {
	var interval time.Duration

	ticker := time.NewTimer(time.Millisecond)
	hasFailed := 0

	for {
		<-ticker.C
		t, err := tm.fetch()
		if err != nil {
			hasFailed++
			interval = time.Duration(hasFailed*hasFailed) * time.Second
		} else {
			tm.token.Store(t.AccessToken)
			interval = time.Duration(t.Expires) * time.Second
		}
		ticker.Reset(interval)
	}
}

type token struct {
	AccessToken string `json:"access_token"`
	Expires     int    `json:"expires_in"`
}

func (tm *tokenManager) fetch() (token, error) {
	var tokenResp token

	resp, err := tm.client.Get(tm.url)
	if err != nil {
		return tokenResp, fmt.Errorf("fail to send req for access token, %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return tokenResp, fmt.Errorf("fail to decode req for access token, %w", err)
	}
	return tokenResp, nil
}

func (tm *tokenManager) Token() (string, error) {
	return tm.token.Load().(string), nil
}
