package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Port           string `mapstructure:"PORT"`
	AuthSvcUrl     string `mapstructure:"AUTH_SVC_URL"`
	AllowedOrigins string `mapstructure:"ALLOWED_ORIGINS"`
	REDIS_ADDR     string `mapstructure:"REDIS_ADDR"`
	UserSvcUrl     string `mapstructure:"USER_SVC_URL"`
	CourseSvcUrl   string `mapstructure:"COURSE_SVC_URL"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	// ВАЖНО: Явно биндим
	viper.BindEnv("PORT")
	viper.BindEnv("AUTH_SVC_URL")
	viper.BindEnv("ALLOWED_ORIGINS")
	viper.BindEnv("REDIS_ADDR")
	viper.BindEnv("USER_SVC_URL")
	viper.BindEnv("COURSE_SVC_URL")

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return
		}
	}

	err = viper.Unmarshal(&config)
	return
}
