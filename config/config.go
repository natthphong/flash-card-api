package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppCode           string
	CompanyCode       string
	Env               string
	JwtAuthConfig     JwtAuthConfig
	Server            Server
	LogConfig         LogConfig
	DBConfig          DBConfig
	HTTP              HTTP
	RedisConfig       RedisConfig
	HomeProxyAdapter  AdapterConfig
	HomeServerAdapter AdapterConfig
}

type JwtAuthConfig struct {
	JwtSecret            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

type RedisConfig struct {
	Mode            string
	Host            string
	Port            string
	Password        string
	DB              int
	PoolTimeout     time.Duration
	DialTimeout     time.Duration
	WriteTimeout    time.Duration
	ReadTimeout     time.Duration
	ConnMaxIdleTime time.Duration
	Cluster         struct {
		Password string
		Addr     []string
	}
}

type Server struct {
	Name string
	Port string
}

type LogConfig struct {
	Level string
}

type DBConfig struct {
	Host            string
	Port            string
	Username        string
	Password        string
	Name            string
	MaxOpenConn     int32
	MaxConnLifeTime int64
}

type HTTP struct {
	TimeOut            time.Duration
	MaxIdleConn        int
	MaxIdleConnPerHost int
	MaxConnPerHost     int
}

type AdapterConfig struct {
	BaseURL string
	Timeout time.Duration
}

func InitConfig() (*Config, error) {

	viper.SetDefault("LogConfig.LEVEL", "info")

	configPath, ok := os.LookupEnv("API_CONFIG_PATH")
	if !ok {
		configPath = "./config"
	}

	configName, ok := os.LookupEnv("API_CONFIG_NAME")
	if !ok {
		configName = "config"
	}

	viper.SetConfigName(configName)
	viper.AddConfigPath(configPath)

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("config file not found. using default/env config: " + err.Error())
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var c Config

	err := viper.Unmarshal(&c)
	if err != nil {
		return nil, err
	}

	return &c, nil

}

func InitTimeZone() {
	ict, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		panic(err)
	}
	time.Local = ict
}
