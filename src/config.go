package src

import (
	"encoding/json"
	"fmt"
	"os"
)

const defaultDatabase = "wechat"

type DbConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}

func (dbCfg DbConfig) Dns() string {
	// "username:password@tcp(host:post)/dbname"
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local",
		dbCfg.Username, dbCfg.Password, dbCfg.Host, dbCfg.Port, defaultDatabase)
}

type Config struct {
	DbConfig
	Token     string `json:"token"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

func NewConfigFromFile(path string) (Config, error) {
	var cfg Config
	f, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("fail to read config file, %w", err)
	}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("fail to decode config file, %w", err)
	}
	return cfg, err
}

const DefaultApiToken = "hanzezhentest"
