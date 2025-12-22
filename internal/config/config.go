package config

import (
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/k0kubun/pp"
)

type Config struct {
	Env        string `yaml:"env"`
	HttpServer `yaml:"http_server"`
	Database   `yaml:"database"`
}

type HttpServer struct {
	Address               string        `yaml:"address"`
	ReadTimeout           time.Duration `yaml:"read_timeout"`
	WriteTimeout          time.Duration `yaml:"write_timeout"`
	IdleTimeout           time.Duration `yaml:"idle_timeout"`
	MailInvitationExpired time.Duration `yaml:"mail_invitation_exp"`
	Mail                  Mail          `yaml:"mail"`
	FrontUrl              string        `yaml:front_url`
}

type Database struct {
	Dsn                string        `yaml:"dsn"`
	MaxOpenConnections int           `yaml:"max_open_connections"`
	MaxIdleConnections int           `yaml:"max_idle_connections"`
	MaxIdleTime        time.Duration `yaml:"max_idle_time"`
}

type Mail struct {
	From   string `yaml:"from"`
	ApiKey string `yaml:"api_key"`
}

func New() *Config {
	filePath := os.Getenv("CONFIG_PATH")

	if filePath == "" {
		pp.Fatal("Failed to retrieve config")
	}

	if _, err := os.Stat(filePath); err != nil {
		pp.Fatal("Path doesn't exists", err.Error())
	}

	var config Config

	if err := cleanenv.ReadConfig(filePath, &config); err != nil {
		pp.Fatal("Could't read config", err.Error())
	}

	return &config
}
