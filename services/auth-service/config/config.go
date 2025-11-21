package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	DBHost        string `mapstructure:"DB_HOST"`
	DBPort        string `mapstructure:"DB_PORT"`
	DBUser        string `mapstructure:"DB_USER"`
	DBPassword    string `mapstructure:"DB_PASSWORD"`
	DBName        string `mapstructure:"DB_NAME"`
	RedisAddr     string `mapstructure:"REDIS_ADDR"`
	AccessSecret  string `mapstructure:"ACCESS_SECRET"`
	RefreshSecret string `mapstructure:"REFRESH_SECRET"`
	GRPCPort      string `mapstructure:"GRPC_PORT"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err = viper.ReadInConfig(); err != nil {
		// Если файла нет, не страшно, если есть ENV переменные
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return
		}
	}

	err = viper.Unmarshal(&config)
	return
}
