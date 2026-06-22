package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Environment string `mapstructure:"ENVIRONMENT"`
	ServerPort  string `mapstructure:"SERVER_PORT"`
	DBUrl       string `mapstructure:"DATABASE_URL"`
	RedisUrl    string `mapstructure:"REDIS_URL"`
	JWTSecret   string `mapstructure:"JWT_SECRET"` 
}


func LoadConfig() (Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("SERVER_PORT", "8080")

	var cfg Config
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("No .env file found, relying on environment variables")
	}

	err := viper.Unmarshal(&cfg)
	return cfg, err
}