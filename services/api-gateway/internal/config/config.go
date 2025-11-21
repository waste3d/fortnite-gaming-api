package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Port       string `mapstructure:"PORT"`
	AuthSvcUrl string `mapstructure:"AUTH_SVC_URL"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	err = viper.Unmarshal(&config)
	return
}
