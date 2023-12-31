package src

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

const (
	defaultDatabase = "wechat"

	defaultTokenFile = "/usr/app/token.json"
)

type DbConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}

func (dbCfg DbConfig) Dns() string {
	// "username:password@tcp(host:post)/dbname"
	dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local",
		dbCfg.Username, dbCfg.Password, dbCfg.Host, dbCfg.Port, defaultDatabase)
	logrus.Infof("dns: %s", dns)
	return dns
}

type Config struct {
	DbConfig
	Token         string `json:"token"`
	AppID         string `json:"app_id"`
	AppSecret     string `json:"app_secret"`
	TokenFilePath string `json:"token_file_path"`
}

func NewConfigFromFile(path string) (Config, error) {
	var cfg Config
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("fail to read config file, %w", err)
	}
	logrus.Infof("config: %s", string(raw))

	if err := json.NewDecoder(bytes.NewBuffer(raw)).Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("fail to decode config file, %w", err)
	}
	if cfg.TokenFilePath == "" {
		cfg.TokenFilePath = defaultTokenFile
	}
	return cfg, err
}

const DefaultApiToken = "hanzezhentest"
