package wechat

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/hanzezhenalex/wechat/src"
)

const (
	failInterval = 5 * time.Second
)

var tracer = logrus.WithField("comp", "token_mngr")

type tokenManager struct {
	token atomic.Value // string

	client *http.Client
	url    string
	path   string
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
		path: cfg.TokenFilePath,
	}
	tm.startLoop()
	return tm
}

func (tm *tokenManager) startLoop() {
	interval := time.Millisecond
	token, err := tm.readTokenFile()

	if err == nil && token.valid() {
		interval = token.nextRefreshInterval()
		tm.token.Store(token.token())
	}

	go tm.daemon(interval)
}

func (tm *tokenManager) daemon(interval time.Duration) {
	ticker := time.NewTimer(interval)

	for {
		<-ticker.C
		tracer.Info("start to fetch token")

		resp, err := tm.fetch()
		tracer.Debugf("token resp: %#v", resp)

		if err != nil {
			interval = failInterval
			tracer.Errorf("fail to fetch token, err=%s, waiting interval=%s",
				err.Error(), interval.String())
		} else {
			token := Token{
				AccessToken:     resp.AccessToken,
				ExpireTimestamp: time.Now().Add(time.Duration(resp.Expires)),
			}
			if token.valid() {
				tm.token.Store(resp.AccessToken)
				go func() {
					if err := tm.writeTokenFile(token); err != nil {
						tracer.Errorf("fail to write token file, %s", err.Error())
					}
				}()

				interval = time.Duration(resp.Expires/2) * time.Second
				tracer.Infof("fetch token successfully, next interval=%s", interval.String())
			} else {
				interval = failInterval
				tracer.Errorf("invalid token fetched, %#v, next interval=%s", token, interval.String())
			}

		}
		ticker.Reset(interval)
	}
}

type TokenResp struct {
	AccessToken string `json:"access_token"`
	Expires     int    `json:"expires_in"`
}

func (tm *tokenManager) fetch() (TokenResp, error) {
	var tokenResp TokenResp

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

type Token struct {
	AccessToken     string    `json:"access_token"`
	ExpireTimestamp time.Time `json:"timestamp"`
}

func (tm *tokenManager) readTokenFile() (Token, error) {
	var token Token
	f, err := os.Open(tm.path)

	if err != nil {
		if os.IsNotExist(err) {
			tracer.Info("token file not exist")
			return token, nil
		} else {
			return token, fmt.Errorf("fail to open token file, %w", err)
		}
	} else {
		if err := json.NewDecoder(f).Decode(&token); err != nil {
			return token, fmt.Errorf("fail to decode token file, %w", err)
		}
	}
	return token, err
}

func (t *Token) valid() bool {
	return time.Now().Before(t.ExpireTimestamp)
}

func (t *Token) nextRefreshInterval() time.Duration {
	interval := t.ExpireTimestamp.Sub(time.Now())
	if interval > time.Minute {
		return interval
	}
	return time.Millisecond
}

func (t *Token) token() string {
	return t.AccessToken
}

func (tm *tokenManager) writeTokenFile(token Token) error {
	f, err := os.OpenFile(tm.path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("fail to open token file, %w", err)
	}
	if err := json.NewEncoder(f).Encode(&token); err != nil {
		return fmt.Errorf("fail to encode token file, %w", err)
	}
	return nil
}
