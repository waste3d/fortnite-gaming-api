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
	APIKey        string `mapstructure:"API_KEY"`
	SMTPEmail     string `mapstructure:"SMTP_EMAIL"`
	FrontendURL   string `mapstructure:"FRONTEND_URL"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	// ВАЖНО: Явно биндим переменные, чтобы Viper их видел без файла
	viper.BindEnv("DB_HOST")
	viper.BindEnv("DB_PORT")
	viper.BindEnv("DB_USER")
	viper.BindEnv("DB_PASSWORD")
	viper.BindEnv("DB_NAME")
	viper.BindEnv("REDIS_ADDR")
	viper.BindEnv("ACCESS_SECRET")
	viper.BindEnv("REFRESH_SECRET")
	viper.BindEnv("GRPC_PORT")
	viper.BindEnv("API_KEY")
	viper.BindEnv("SMTP_EMAIL")
	viper.BindEnv("FRONTEND_URL")

	// Пытаемся прочитать файл, но не умираем, если его нет
	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return
		}
		// Файла нет? Ну и ладно, работаем на ENV
	}

	err = viper.Unmarshal(&config)
	return
}
